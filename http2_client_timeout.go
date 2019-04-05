// https://github.com/golang/go/issues/21978
//
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"golang.org/x/net/http2"
)

func main() {
	fail := false
	flag.BoolVar(&fail, "fail", false, "enable failure behavior (use an HTTP/2 client)")
	flag.Parse()

	// Start an HTTP/2 server.
	handler := func(rw http.ResponseWriter, req *http.Request) {
		log.Println(req.Method, req.RequestURI, req.Proto)
		rw.WriteHeader(http.StatusOK)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	srv.TLS = &tls.Config{
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		NextProtos:   []string{http2.NextProtoTLS},
	}
	srv.StartTLS()
	err := http2.ConfigureServer(srv.Config, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare to drop packets from the server.
	ipt, err := iptables.New()
	if err != nil {
		log.Fatal("iptables: ", err)
	}
	u, err := url.Parse(srv.URL)
	if err != nil {
		log.Fatal(err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		log.Fatal(err)
	}
	rule := []string{"--source", host, "-p", "tcp", "--sport", port, "-j", "DROP"}

	// Initialize an HTTP client with custom dialer and transport.
	client := srv.Client()
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Restore connectivity whenever a Dial occurs.
		ipt.Delete("filter", "OUTPUT", rule...)

		// Dial normally and print the results.
		conn, err := (&net.Dialer{
			KeepAlive: 1 * time.Second,
		}).DialContext(ctx, network, addr)
		if err != nil {
			log.Println("dial: err", err)
		} else {
			log.Println("dial: dst", addr, "src", conn.LocalAddr())
		}
		return conn, err
	}
	transport := &http.Transport{
		DialContext:     dial,
		MaxIdleConns:    1,
		TLSClientConfig: client.Transport.(*http.Transport).TLSClientConfig,
	}
	// Enable HTTP/2 to trigger permanent timeouts.
	if fail {
		err = http2.ConfigureTransport(transport)
		if err != nil {
			log.Fatal(err)
		}
	}
	client.Transport = transport
	client.Timeout = 1 * time.Second

	// Perform a GET request to establish a connection.
	doit := func() bool {
		resp, err := client.Get(srv.URL)
		if err != nil {
			log.Println(err)
			return false
		} else {
			log.Println(resp.Status)
			return true
		}
	}
	doit()

	// Drop packets from the server, until the next dial or exit.
	err = ipt.Insert("filter", "OUTPUT", 1, rule...)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ipt.Delete("filter", "OUTPUT", rule...)
	}()

	// Attempt to perform a few more GET requests. If HTTP/2 is enabled all
	// subsequent requests will fail and the connection will remain open.
	//
	// Otherwise, the connection will be closed after the first timeout,
	// the dial function will be called, and a new connection established.
	for !doit() {
	}
}
