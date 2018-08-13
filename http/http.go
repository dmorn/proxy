package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tecnoporto/proxy/dialer"
	"github.com/tecnoporto/proxy/transmit"
)

// Proxy represents a HTTP proxy server implementation.
type Proxy struct {
	dialer.Dialer
	port int

	s *http.Server
	c *http.Client
}

// New returns a new Proxy instance that creates connections using the
// default net.Dialer, if d is nil. Otherwise it creates the proxy using
// the dialer passed.
func New(d *dialer.Dialer) *Proxy {
	var _d dialer.Dialer = dialer.Default
	if d != nil {
		_d = *d
	}

	p := new(Proxy)
	p.Dialer = _d
	p.s = &http.Server{
		Handler:        p,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	p.c = &http.Client{Transport: tr}

	return p
}

func (p *Proxy) ListenAndServe(ctx context.Context, port int) error {
	c := make(chan error)
	go func() {
		p.s.Addr = fmt.Sprintf(":%d", port)
		c <- p.s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		p.s.Shutdown(context.Background())
		<-c
		return ctx.Err()
	case err := <-c:
		return err
	}
}

func (p *Proxy) Protocol() string {
	return "http"
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := p.c.Transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()

	if err := copyHeader(w.Header(), resp.Header); err != nil {
		// TODO: correct status code
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	// copy data
	io.Copy(w, resp.Body)
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	// create remote connection
	dst_conn, err := p.DialContext(context.Background(), "tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	// take over source connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Proxy is not able to take over source connection. Hijacking is not supported", http.StatusInternalServerError)
		return
	}

	src_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// copy data
	ctx := transmit.NewContext(context.Background(), time.Second*30, 1500)
	if err = transmit.Data(ctx, dst_conn, src_conn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func copyHeader(dst, src http.Header) error {
	for k, v := range src {
		dst[k] = v
	}
	return nil
}
