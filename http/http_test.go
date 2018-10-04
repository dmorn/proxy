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

package http_test

import (
	"net/http"
	"testing"

	proxy_http "github.com/tecnoporto/proxy/http"
)

func TestCleanHeader(t *testing.T) {
	h := http.Header(make(map[string][]string))
	connK := http.CanonicalHeaderKey("Connection")
	fooK := http.CanonicalHeaderKey("Foo")

	h.Add(fooK, "some value")
	h.Add(connK, fooK)

	proxy_http.CleanHeader(&h)

	if s := h.Get(fooK); s != "" {
		t.Fatalf("field %v should be empty, found %v instead", fooK, s)
	}
	if s := h.Get(connK); s != "" {
		t.Fatalf("field %v should be empty, found %v instead", fooK, s)
	}
}
