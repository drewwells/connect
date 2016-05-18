package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
)

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	fmt.Printf("Starting proxy server at: %s", *addr)
	if *verbose {
		fmt.Println(" with verbose on")
	}
	log.Fatal(http.ListenAndServe(*addr, proxy))
}
