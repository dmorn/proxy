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

package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"

	"github.com/booster-proj/proxy"
	"github.com/booster-proj/proxy/log"
)

var port = flag.Int("port", 1080, "server listening port")
var rawProto = flag.String("proto", "", "proxy protocol used. Available protocols: http, https, socks5")

func main() {
	flag.Parse()

	if *rawProto == "" {
		log.Fatal(errors.New("proto flag is required"))
	}

	proto, err := proxy.ParseProto(*rawProto)
	if err != nil {
		log.Fatal(err)
	}

	var p proxy.Proxy
	switch proto {
	case proxy.HTTP:
		p, err = proxy.NewHTTP()
	case proxy.SOCKS5:
		p, err = proxy.NewSOCKS5()
	default:
		err = errors.New("protocol (" + *rawProto + ") is not yet supported")
	}
	if err != nil {
		log.Fatal(err)
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

	log.Info.Printf("proxy (%v) listening on :%d", p.Protocol(), *port)
	if err := p.ListenAndServe(ctx, *port); err != nil {
		log.Fatal(err)
	}
}
