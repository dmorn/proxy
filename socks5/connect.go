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

package socks5

import (
	"context"
	"errors"
	"net"
)

// connect dials a new connection with target, which must be a canonical
// address with host and port.
func (s *Socks5) Connect(ctx context.Context, conn net.Conn, target string) (net.Conn, error) {
	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version)

	tconn, err := s.DialContext(ctx, "tcp", target)
	if err != nil {
		// TODO(daniel): Respond with proper code
		buf = append(buf, socks5RespHostUnreachable, socks5FieldReserved)
		if _, err := conn.Write(buf); err != nil {
			return nil, errors.New("Connect: unable to write connect response: " + err.Error())
		}

		return nil, err
	}
	// BUG: sometimes there is no err BUT the connection is nil
	if tconn == nil {
		return nil, errors.New("Connect: Dial returned nil connection")
	}

	buf = append(buf, socks5RespSuccess, socks5FieldReserved)

	// bnd addr
	addr := tconn.LocalAddr().(*net.TCPAddr)
	ip := addr.IP
	port := uint16(addr.Port)

	if ip4 := ip.To4(); ip4 != nil {
		buf = append(buf, socks5IP4)
		ip = ip4
	} else {
		buf = append(buf, socks5IP6)
	}
	buf = append(buf, ip...)

	// bdn port
	buf = append(buf, byte(port>>8), byte(port)&0xff)

	conn.Write(buf)

	return tconn, nil
}
