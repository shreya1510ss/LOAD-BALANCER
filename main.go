package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

type Server struct{
	address string
	weight int
	connections int64
	healthy int32
}

type ServerPool struct {
	servers sync.Map
}

func (sp *ServerPool) AddServer(address string, weight int){
      sp.servers.Store(address,&Server{
		 address: address,
		 weight: weight,
		 connections: 0,
		 healthy: 1,
	  })
} 

func (sp * ServerPool) RemoverServer(address string){
      sp.servers.Delete(address)
}

func (sp *ServerPool) GetServers() []*Server {
    servers := make([]*Server, 0)
    sp.servers.Range(func(key, value interface{}) bool {
        server := value.(*Server)
        if atomic.LoadInt32(&server.healthy) == 1 {
            servers = append(servers, server)
        }
        return true
    })
    return servers
}



func LeastConnections(servers []*Server, _ *http.Request) *Server {
	var bestServer *Server
	for _, server := range servers {
		if bestServer == nil || server.connections < bestServer.connections {
			bestServer = server
		}
	}
	return bestServer
}

var rrCounter uint64

func RoundRobin(servers []*Server, _ *http.Request) *Server {
	idx := atomic.AddUint64(&rrCounter, 1) - 1
	return servers[idx%uint64(len(servers))]
}

func IPHash(servers []*Server, r *http.Request) *Server {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	h := fnv.New32a()
	h.Write([]byte(ip))
	return servers[h.Sum32()%uint32(len(servers))]
}

func healthCheck(server *Server) bool {
	u, err := url.Parse("http://" + server.address + "/health")
	if err != nil {
		return false
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(u.String())
	if err != nil {
		return false
	}

	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}


func runHealthCheck(sp *ServerPool, interval time.Duration) {
    for {
        time.Sleep(interval)
        sp.servers.Range(func(addr, server interface{}) bool {
            s := server.(*Server)
            if healthCheck(s) {
                atomic.StoreInt32(&s.healthy, 1)
            } else {
                atomic.StoreInt32(&s.healthy, 0)
            }
            return true
        })
    }
}



type LoadBalancer struct {
	pool      *ServerPool
	algorithm func([]*Server, *http.Request) *Server
	interval  time.Duration
}

func(lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request){
	    servers := lb.pool.GetServers()
		if len(servers) == 0 {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		server := lb.algorithm(servers, r)
		if server == nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		atomic.AddInt64(&server.connections, 1)
		defer atomic.AddInt64(&server.connections, -1)

		proxy := httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host: server.address,
		})

		proxy.ServeHTTP(w, r)



	}





func main() {

	pool := &ServerPool{}
	algorithm := RoundRobin
	interval := 10 * time.Second
	
	pool.AddServer(getEnv("SERVER1_ADDR", "localhost:8081"), 1)
	pool.AddServer(getEnv("SERVER2_ADDR", "localhost:8082"), 1)
	pool.AddServer(getEnv("SERVER3_ADDR", "localhost:8083"), 1)

	lb := &LoadBalancer{
		pool:      pool,
		algorithm: algorithm,
		interval:  interval,
	}

	go runHealthCheck(pool, interval)

	http.Handle("/", lb)
	fmt.Println("Load Balancer is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))

}