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

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"errors"

	"github.com/tecnoporto/proxy"
)

var port = flag.Int("port", 1080, "server listening port")
var rawProto = flag.String("proto", "", "proxy protocol used. Available protocols: http, socks5")

var logger = log.New(os.Stdout, "", log.LstdFlags | log.Lshortfile)

func main() {
	flag.Parse()

	if *rawProto == "" {
		logger.Fatal(errors.New("proto flag is required"))
	}

	proto, err := proxy.ParseProto(*rawProto)
	if err != nil {
		logger.Fatal(err)
	}

	p, err := proxy.New(proto, nil)
	if err != nil {
		logger.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("proxy (%v) listening on :%d", p.Protocol(), *port)
	if err := p.ListenAndServe(ctx, *port); err != nil {
		log.Fatal(err)
	}
}
