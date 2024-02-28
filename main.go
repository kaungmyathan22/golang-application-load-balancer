package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

type ApplicationServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		servers:         servers,
		roundRobinCount: 0,
	}
}

func NewApplicationServer(addr string) *ApplicationServer {
	serverURL, err := url.Parse(addr)
	handleError(err)
	return &ApplicationServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}

}

func main() {
	servers := []Server{
		NewApplicationServer("https://google.com"),
		NewApplicationServer("https://bing.com"),
		NewApplicationServer("https://duckduckgo.com"),
	}
	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Printf("serving request at localhost:%s\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (server *ApplicationServer) IsAlive() bool {
	return true
}

func (server *ApplicationServer) Address() string {
	return server.addr
}

func (server *ApplicationServer) Serve(rw http.ResponseWriter, r *http.Request) {
	server.proxy.ServeHTTP(rw, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, r)
}
