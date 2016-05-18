package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
	Remote  string
}

func main() {
	var listen string
	p := params{}
	c := config{Listen: ":7999"}

	flag.BoolVar(&p.verbose, "v", false, "should every proxy request be logged to stdout")
	flag.StringVar(&p.configFile, "config", "", "location of config file")
	flag.StringVar(&listen, "listen", "", "address to listen to")
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

	if len(listen) > 0 {
		c.Listen = listen
	}

	serve(c, p.verbose)

}

func rules(c config, l *url.URL) bool {
	path := l.Host
	var forward bool
	for _, allow := range c.Allow {
		if strings.Contains(path, allow) {
			forward = true
		}

	}
	if forward {
		for _, block := range c.Block {
			if path == block {
				forward = false
			}
		}
	}
	if forward {
		fmt.Println("forwarding request to proxy", path)
	}
	return forward
}

func serve(c config, verbose bool) {

	svr := goproxy.NewProxyHttpServer()
	svr.Verbose = verbose
	fmt.Printf("...Starting proxy server at: %s verbose: %t\n", c.Listen, verbose)
	remoteaddr := c.Remote
	remoteURL, err := url.Parse(remoteaddr)
	if err != nil {
		log.Fatal("failed to parse upstream server:", err)
	}
	fmt.Println("upstream server", remoteaddr)

	svr.Tr = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			if !rules(c, r.URL) {
				return nil, nil
			}

			return remoteURL, nil
		},
	}

	log.Fatal(http.ListenAndServe(c.Listen, svr))
}
