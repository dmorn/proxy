package proxy

import (
	"context"
	"io"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

type key int

const idleTimeoutKey key = 0

// NewContext returns a context that stores the idleTimeout inside
// its Value field.
func NewContext(ctx context.Context, idleTimeout time.Duration) context.Context {
	return context.WithValue(ctx, idleTimeoutKey, idleTimeout)
}

// FromContext extracts the idleTimeout from the context.
func FromContext(ctx context.Context) (time.Duration, bool) {
	return ctx.Value(idleTimeoutKey).(time.Duration)
}

// DefaultIdleTimeout returns the default duration of 5 seconds used
// to determine when to close an idle connection.
func DefaultIdleTimeout() (d time.Duration) {
	d, _ = time.ParseDuration("5s")
	return
}

// ProxyData copies data from src to dst and the other way around.
// Closes the connections when no data is transferred for a defined duration, i.e.
// the idleTimeout value stored in the context or the DefaultIdleTimeout, if the
// former is not present.
func ProxyData(ctx context.Context, src net.Conn, dst net.Conn) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.proxyData(ctx, src, dst)
	})
	g.Go(func() error {
		return s.proxyData(ctx, dst, src)
	})

	return g.Wait()
}

func proxyData(ctx context.Context, src net.Conn, dst net.Conn) error {
	idle := DefaultIdleTimeout()
	if d, ok := FromContext(ctx); ok {
		idle = d
	}

	errc := make(chan error, 1)
	done := func() {
		src.Close()
		dst.Close()
	}
	timer := time.AfterFunc(idle, done)

	go func() {
		for {
			_, err := io.CopyN(src, dst, s.ChunkSize)
			errc <- err
		}
	}()

	for {
		select {
		case <-ctx.Done():
			done()
			timer.Stop()
			close(errc)

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
