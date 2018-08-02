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

// Package proxy provides utility functions to proxy data through network
// connections, with control over timeouts.
package proxy

import (
	"context"
	"io"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

type key int

const (
	idleTimeoutKey key = iota
	transmittingUnitKey
)

// DefaultIdleTimeout is the default duration of 5 minutes used
// to determine when to close an idle connection.
const DefaultIdleTimeout time.Duration = time.Minute * 5

// DTU is the default trasmitting unit used when copying data
// from one connection to another.
const DTU int64 = 1500

// NewContext returns a context that stores the idleTimeout inside
// its Value field.
func NewContext(ctx context.Context, idleTimeout time.Duration, tu int64) context.Context {
	ctx = context.WithValue(ctx, idleTimeoutKey, idleTimeout)
	return context.WithValue(ctx, transmittingUnitKey, tu)
}

// DurationFromContext extracts the idleTimeout from the context.
func DurationFromContext(ctx context.Context) (time.Duration, bool) {
	d, ok := ctx.Value(idleTimeoutKey).(time.Duration)
	return d, ok
}

// TUFromContext extracts the transmitting unit from context.
func TUFromContext(ctx context.Context) (int64, bool) {
	i, ok := ctx.Value(transmittingUnitKey).(int64)
	return i, ok
}

// Data copies data from src to dst and the other way around.
// Closes the connections when no data is transferred for a defined duration, i.e.
// the idleTimeout value stored in the context or the DefaultIdleTimeout, if the
// former is not present.
func Data(ctx context.Context, src net.Conn, dst net.Conn) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return proxyData(ctx, src, dst)
	})
	g.Go(func() error {
		return proxyData(ctx, dst, src)
	})

	return g.Wait()
}

func proxyData(ctx context.Context, src net.Conn, dst net.Conn) error {
	// allowed idle timeout before closing the connection.
	idle := DefaultIdleTimeout
	if d, ok := DurationFromContext(ctx); ok {
		idle = d
	}

	// transimitting unit
	tu := DTU
	if i, ok := TUFromContext(ctx); ok {
		tu = i
	}

	errc := make(chan error, 1)

	done := func() {
		src.Close()
		dst.Close()
	}
	timer := time.AfterFunc(idle, done)

	go func() {
		for {
			_, err := io.CopyN(src, dst, tu)
			errc <- err
		}
		close(errc)
	}()

	for {
		select {
		case <-ctx.Done():
			done()
			timer.Stop()

			return ctx.Err()
		case err := <-errc:
			if err != nil {
				return err
			}

			// io operations did not return any errors. Reset
			// deadline and keep on transferring data
			timer.Reset(idle)
		}

	}
}
