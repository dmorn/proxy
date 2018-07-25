package main

import (
	"context"
	"flag"
	"log"

	"github.com/tecnoporto/proxy/socks5"
)

var port = flag.Int("port", 1080, "server listening port")

func main() {
	flag.Parse()

	p := socks5.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.ListenAndServe(ctx, *port); err != nil {
		log.Fatal(err)
	}
}
