package http

import (
	"context"
	"errors"

	"github.com/tecnoporto/proxy/dialer"
)

// Proxy represents a HTTP proxy server implementation.
type Proxy struct {
	dialer.Dialer
	port int
}

// New returns a new Proxy instance that creates connections using the
// default net.Dialer, if d is nil. Otherwise it creates the proxy using
// the dialer passed.
func New(d *dialer.Dialer) *Proxy {
	var _d dialer.Dialer = dialer.Default
	if d != nil {
		_d = *d
	}

	return &Proxy{
		Dialer: _d,
	}
}

func (p *Proxy) ListenAndServe(ctx context.Context, port int) error {
	return errors.New("http proxy not yet implemented")
}

func (p *Proxy) Protocol() string {
	return "http"
}

