package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/hashicorp/hcl"
)

type params struct {
	verbose    bool
	configFile string
}

// config defines the allowed parameters read from a file.
type config struct {
	Forward string   // addr of proxy aware socks5 server
	Listen  string   // addr to listen on
	Allow   []string // Slice of patterns to forward
	Block   []string
}

func main() {
	p := params{}
	c := config{}

	flag.BoolVar(&p.verbose, "v", false, "should every proxy request be logged to stdout")
	flag.StringVar(&p.configFile, "config", "", "location of config file")
	flag.StringVar(&c.Listen, "listen", ":7999", "address to listen to")
	flag.Parse()

	if len(p.configFile) > 0 {
		bs, err := ioutil.ReadFile(p.configFile)
		if err == nil {
			err = hcl.Decode(&c, string(bs))
			if err != nil {
				log.Fatalf("invalid hcl file: %s", err)
			}
		}
	}

	serve(c, p.verbose)

}

func serve(c config, verbose bool) {

	svr := goproxy.NewProxyHttpServer()
	svr.Verbose = verbose
	fmt.Printf("Starting proxy server at: %s verbose: %t", c.Listen, verbose)

	var allow goproxy.ReqConditionFunc = func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		for _, al := range c.Allow {
			if strings.Contains(req.URL.String(), al) {
				return true
			}
		}
		return false
	}

	svr.OnRequest(
		allow,
		// goproxy.ReqHostIs(c.Allow...),
		goproxy.Not(goproxy.ReqHostIs(c.Block...)),
	).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		fmt.Println("proxying", req.URL)
		return req, nil
	})

	log.Fatal(http.ListenAndServe(c.Listen, svr))
}
