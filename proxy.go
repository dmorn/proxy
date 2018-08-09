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

// Package proxy provides wrappers around various proxy server
// implementations.
package proxy

import (
	"context"
	"fmt"

	"github.com/tecnoporto/proxy/http"
	"github.com/tecnoporto/proxy/socks5"
	"github.com/tecnoporto/proxy/dialer"
)

// Proxy explains how a proxy should behave.
type Proxy interface {
	// Protocol is the string representation of the protocol
	// beign used.
	Protocol() string
	// ListenAndServe reveleas the proxy to the network.
	ListenAndServe(ctx context.Context, port int) error
}

// Protocol is a wrapper around uint8.
type Protocol uint8

// Proxy protocol available.
const (
	HTTP Protocol = iota
	SOCKS5
	Unknown
)

// ParseProto takes a string as input and returns its Protocol value,
// is one is recognised. Otherwise it returns an error.
func ParseProto(s string) (Protocol, error) {
	switch s {
	case "http", "HTTP":
		return HTTP, nil
	case "socks5", "SOCKS5":
		return SOCKS5, nil
	default:
		return Unknown, fmt.Errorf("unrecognised proto: %s", s)
	}
}

// New returns a new proxy instance that speaks the protocol assigned.
// d can also be nil, in that case the proxy will use a default dialer,
// usually a bare net.Dialer.
func New(p Protocol, d *dialer.Dialer) (Proxy, error) {
	switch p {
	case HTTP:
		return http.New(d), nil
	case SOCKS5:
		return socks5.New(d), nil
	default:
		return nil, fmt.Errorf("unrecognised protocol: %v", p)
	}
}

