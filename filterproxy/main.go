package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"sync"

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

	// pprof
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

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
			// l.Host has port number on the end
			if strings.HasPrefix(path, block) {
				forward = false
			}
		}
	}
	if forward {
		// fmt.Println("forwarding request to proxy", path)
	}

	seenMu.Lock()
	if _, ok := seen[path]; !ok {
		fmt.Println("addr ", forward, path)
		seen[path] = struct{}{}
	}
	seenMu.Unlock()

	return forward
}

var seenMu = sync.RWMutex{}
var seen = map[string]struct{}{}

// FIXME: make it part of config
var printVerbose bool

func printf(format string, a ...interface{}) {
	if printVerbose {
		fmt.Printf(format, a...)
	}
}

func serve(c config, verbose bool) {
	printVerbose = verbose
	svr := goproxy.NewProxyHttpServer()
	svr.Verbose = verbose
	fmt.Printf("Starting proxy server at: %s verbose: %t\n", c.Listen, verbose)
	remoteaddr := c.Remote
	remoteURL, err := url.Parse(remoteaddr)
	if err != nil {
		log.Fatal("failed to parse upstream server:", err)
	}
	printf("upstream server %s\n", remoteaddr)

	dial := func(network, addr string) (net.Conn, error) {
		u := &url.URL{Host: addr}
		// non-HTTP traffic seems to not use proxy, check again here
		// ensuring we aren't preventin access to the remote address
		if !rules(c, u) {
			return net.Dial(network, addr)
		}
		printf("dialing upstream proxy... %s\n", addr)
		return svr.NewConnectDialToProxy(remoteaddr)(network, addr)
	}
	_ = remoteURL
	svr.Tr = &http.Transport{
		DialTLS: dial,
		Dial:    dial,
		// Proxy: func(r *http.Request) (*url.URL, error) {
		// 	if !rules(c, r.URL) {
		// 		printf("Proxy Bypass %s\n", r.URL)
		// 		return r.URL, nil
		// 	}
		// 	printf("Forwarding request for %s\n", r.URL)
		// 	return remoteURL, nil
		// },
		// All requests go to the same host, this will increase throughput
		MaxIdleConnsPerHost: 1000,
		DisableKeepAlives:   true,
	}

	log.Fatal(http.ListenAndServe(c.Listen, svr))
}
