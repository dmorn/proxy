/*
MIT License

Copyright (c) 2018 tecnoporto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Package http provides a http proxy implementation.
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tecnoporto/proxy/dialer"
	"github.com/tecnoporto/proxy/transmit"
)

var logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

// Proxy represents a HTTP proxy server implementation.
type Proxy struct {
	dialer.Dialer
	port int

	s *http.Server
	c *http.Client

	tls *tlsConfig
}

type tlsConfig struct {
	certPath string
	keyPath  string
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
		TLSNextProto:   make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	p.c = &http.Client{Transport: tr}

	return p
}

func NewTLS(d *dialer.Dialer, cert, key string) *Proxy {
	p := New(d)
	p.tls = &tlsConfig{
		certPath: cert,
		keyPath:  key,
	}

	return p
}

func (p *Proxy) ListenAndServe(ctx context.Context, port int) error {
	c := make(chan error)
	go func() {
		p.s.Addr = fmt.Sprintf(":%d", port)

		if p.tls == nil {
			c <- p.s.ListenAndServe()
		} else {
			c <- p.s.ListenAndServeTLS(p.tls.certPath, p.tls.keyPath)
		}
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
	if p.tls == nil {
		return "http"
	} else {
		return "https"
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Cleanup header fields to relvant to the upstream
	CleanHeader(&r.Header)

	resp, err := p.c.Transport.RoundTrip(r)
	if err != nil {
		logger.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()

	CleanHeader(&resp.Header)
	CopyHeader(w.Header(), resp.Header)

	w.WriteHeader(http.StatusOK)

	io.Copy(w, resp.Body)
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	// create remote connection
	dst_conn, err := p.DialContext(context.Background(), "tcp", r.Host)
	if err != nil {
		logger.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	// take over source connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		mess := "Proxy is not able to take over source connection. Hijacking is not supported"
		logger.Println(mess)
		http.Error(w, mess, http.StatusInternalServerError)
		return
	}

	src_conn, _, err := hijacker.Hijack()
	if err != nil {
		logger.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// copy data from src_ to dst_conn and vice versa
	ctx := transmit.NewContext(context.Background(), time.Second*30, 1500)
	transmit.Data(ctx, dst_conn, src_conn)
}

func CopyHeader(dst, src http.Header) {
	for k, v := range src {
		dst[k] = v
	}
}

// CleanHeader cleans the header from the fields that are not intended to
// be relevant to downstream recipients. See RFC 7230, 6.1.
func CleanHeader(h *http.Header) {
	k := "Connection"
	v := h.Get(k)

	h.Del(v) // delete the field referenced by the Connection field
	h.Del(k) // delete the Connection field itself
}
