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

package socks5_test

import (
	"testing"

	"github.com/tecnoporto/proxy/socks5"
)

func TestNegotiate(t *testing.T) {
	s5 := new(socks5.Proxy)
	conn := new(conn)

	var tests = []struct {
		in  []byte
		out []byte
		err bool // should expect negotiation error?
	}{
		{in: []byte{5, 2, 0, 1}, out: []byte{5, 0}, err: false}, // successful response
		{in: []byte{5, 1, 1}, out: []byte{5, 0xff}, err: false}, // command not supported
		{in: []byte{4}, out: []byte{}, err: true},               // wrong version
		{in: []byte{5, 0, 1}, out: []byte{}, err: true},         // wrong methods number
	}

	for _, test := range tests {
		if _, err := conn.Write(test.in); err != nil {
			t.Fatal(err)
		}

		if err := s5.Negotiate(conn); err != nil {
			// only fail if not expecting any error
			if !test.err {
				t.Fatal(err)
			} else {
				return
			}
		}

		buf := make([]byte, 2)
		if _, err := conn.Read(buf); err != nil {
			t.Fatal(err)
		}

		for i, v := range buf {
			if v != test.out[i] {
				t.Fatalf("unexpected result. Wanted %v, found %v", test.out, buf)
			}
		}

	}
}
