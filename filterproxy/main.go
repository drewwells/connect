package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"

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
	fmt.Printf("...Starting proxy server at: %s verbose: %t\n", c.Listen, verbose)
	remoteaddr := ":6543"
	fmt.Println("upstream server", remoteaddr)

	tr := func(network, addr string) (net.Conn, error) {
		fmt.Println("tr...", addr)
		// return svr.NewConnectDialToProxy(remoteaddr)(network, addr)
		// return svr.ConnectDial("tcp", remoteaddr)
		//fmt.Println("sending upstream ", remoteaddr)
		return net.Dial("tcp", addr)
	}
	_ = tr
	// svr.Tr.Dial = tr
	// svr.Tr.DialTLS = tr

	// svr.ConnectDial = func(net, addr string) (net.Conn, error) {
	// 	fmt.Println("connectdial to upstream", addr)
	// 	return svr.NewConnectDialToProxy(addr)(net, addr)
	// }
	//svr.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*baidu.com$"))).
	//	HandleConnect(goproxy.AlwaysReject)
	svr.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		HandleConnect(goproxy.AlwaysMitm)

	svr.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*:80$"))).

		// svr.OnRequest(
		// 	goproxy.ReqConditionFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		// 		for _, al := range c.Allow {
		// 			if strings.Contains(req.URL.String(), al) {
		// 				return true
		// 			}
		// 		}
		// 		return false
		// 	}),
		// 	//goproxy.Not(goproxy.ReqHostIs(c.Block...)),
		// )
		HijackConnect(func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
			defer func() {
				if e := recover(); e != nil {
					ctx.Logf("error connecting to remote: %v", e)
					client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
				}
				client.Close()
			}()
			fmt.Println("Hijacking", req.URL)
			clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))
			remote, err := connectDial(svr, "tcp", req.URL.Host)
			orPanic(err)
			remoteBuf := bufio.NewReadWriter(bufio.NewReader(remote), bufio.NewWriter(remote))
			for {
				req, err := http.ReadRequest(clientBuf.Reader)
				orPanic(err)
				orPanic(req.Write(remoteBuf))
				orPanic(remoteBuf.Flush())
				resp, err := http.ReadResponse(remoteBuf.Reader, req)
				orPanic(err)
				orPanic(resp.Write(clientBuf.Writer))
				orPanic(clientBuf.Flush())
			}
		})
		// DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		// 	addr := ":6543"
		// 	fmt.Printf("proxying %s to %s\n", req.URL, addr)
		// 	return req, nil
		// })

	log.Fatal(http.ListenAndServe(c.Listen, svr))
}

func orPanic(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// copied/converted from https.go
func connectDial(proxy *goproxy.ProxyHttpServer, network, addr string) (c net.Conn, err error) {
	if proxy.ConnectDial == nil {
		return dial(proxy, network, addr)
	}
	return proxy.ConnectDial(network, addr)
}

// copied/converted from https.go
func dial(proxy *goproxy.ProxyHttpServer, network, addr string) (c net.Conn, err error) {
	if proxy.Tr.Dial != nil {
		return proxy.Tr.Dial(network, addr)
	}
	return net.Dial(network, addr)
}
