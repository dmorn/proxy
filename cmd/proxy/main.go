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
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/tecnoporto/proxy"
)

var port = flag.Int("port", 1080, "server listening port")
var rawProto = flag.String("proto", "", "proxy protocol used. Available protocols: http, https, socks5")

var cert = flag.String("cert", "server.pem", "path to cert file")
var key = flag.String("key", "server.key", "path to key file")

var logger = log.New(os.Stdout, "", log.LstdFlags|log.Llongfile)

func main() {
	flag.Parse()

	if *rawProto == "" {
		logger.Fatal(errors.New("proto flag is required"))
	}

	proto, err := proxy.ParseProto(*rawProto)
	if err != nil {
		logger.Fatal(err)
	}

	var p proxy.Proxy
	switch proto {
	case proxy.HTTP:
		p, err = proxy.NewHTTP(nil)
	case proxy.HTTPS:
		p, err = proxy.NewHTTPS(nil, *cert, *key)
	case proxy.SOCKS5:
		p, err = proxy.NewSOCKS5(nil)
	default:
		err = errors.New("protocol (" + *rawProto + ") is not yet supported")
	}
	if err != nil {
		logger.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	go func() {
		for _ = range c {
			cancel()
		}
	}()

	log.Printf("proxy (%v) listening on :%d", p.Protocol(), *port)
	if err := p.ListenAndServe(ctx, *port); err != nil {
		log.Fatal(err)
	}
}
