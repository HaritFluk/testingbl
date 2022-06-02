package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// This is the LoadBalancer struct
// It has a port, a roundRobinCounter, and a list of servers
// The roundRobinCounter is used to keep track of which server to use next
// The servers is a list of servers that the load balancer will connect to
// The port is the port that the load balancer will listen on
type LoadBalancer struct {
	port 				string
	roundRobinCounter 	int
	servers				[]Server
}

// This is the Server Interface.
type Server interface{
	// Address return the address with which to accesss the server
	Address() string

	// IsAlive returns true if the server is alive and false otherwise
	IsAlive() bool

	// Serve uses this to process requests
	Serve(rw http.ResponseWriter, r *http.Request)
}

// This is the simpleServer struct
// Address is the address of the server
// Proxy is the proxy that the server uses to connect to the server
type simpleServer struct {
	address string
	proxy *httputil.ReverseProxy
}

// This is the ServeHTTP method
// It takes in a request and response and does the following:
func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port: 				port,
		roundRobinCounter: 	0,
		servers: 			servers,
	}
}

func (s *simpleServer) Address() string  { return s.address }

func (s *simpleServer) IsAlive() bool    { return true }

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

// handleErr prints the error and exits the program
// Note: This is not how one would want to handle an error in production, but
// serves well for demonstration purposes.
func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// NewSimpleServer returns a new simpleServer
// The address is the address of the server
// The proxy is the proxy that the server uses to connect to the server
func newSimpleServer(address string) *simpleServer {
	serverUrl, err := url.Parse(address)
	handleErr(err)

	return &simpleServer{
		address: address,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

// getNextAvailableServer returns the address of the next available server to send a
// request to, using a simple round robin algorithm
func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCounter%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCounter++
		server = lb.servers[lb.roundRobinCounter%len(lb.servers)]
	}
	lb.roundRobinCounter++
	
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()

	// could optionally log stuff the requests here!
	fmt.Printf("forwarding request to '%s'\n", targetServer.Address())

	// could delete pre-existing X-Forwarded-For header to prevent IP spoofing
	targetServer.Serve(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.google.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.duckduckgo.com"),
	}

	lb := NewLoadBalancer("8080", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	// Register a proxy handle all requests
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}