/*
MIT License

Copyright (c) 2018 KIM KeepInMind Gmbh/srl

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
	"errors"
	"io"
	"net"
)

// negotiate performs the very first method subnegotiation when handling a new
// connection.
func (s *Proxy) Negotiate(conn net.Conn) error {

	// len is just an estimation
	buf := make([]byte, 7)

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return errors.New("proxy: failed to read negotiation: " + err.Error())
	}

	v := buf[0]         // protocol version
	nm := uint8(buf[1]) // number of methods

	if cap(buf) < int(nm) {
		buf = make([]byte, nm)
	} else {
		buf = buf[:nm]
	}

	// Check version number
	if v != socks5Version {
		return errors.New("proxy: unsupported version: " + string(v))
	}

	if _, err := io.ReadFull(conn, buf[:nm]); err != nil {
		return errors.New("proxy: failed to read methods: " + err.Error())
	}

	// select one method; could also be socksV5MethodNoAcceptableMethods
	m := acceptMethod(buf)

	buf = buf[:0]
	buf = append(buf, socks5Version)
	buf = append(buf, m)

	if _, err := conn.Write(buf); err != nil {
		return errors.New("proxy: unable to write negotitation response: " + err.Error())
	}

	return nil
}

func acceptMethod(m []uint8) uint8 {
	for _, sm := range supportedMethods {
		for _, tm := range m {
			if sm == tm {
				return sm
			}
		}
	}

	return socks5MethodNoAcceptableMethods
}
