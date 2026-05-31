package main

import "sync"

type Server struct{
	address string
	weight int
	connections int64
}

type ServerPool struct {
	servers sync.Map
}

func (sp *ServerPool) AddServer(address string, weight int){
      sp.servers.Store(address,&Server{
		 address: address,
		 weight: weight,
		 connections: 0,
	  })
} 

func (sp * ServerPool) RemoverServer(address string){
      sp.servers.Delete(address)
}

func (sp *ServerPool) GetServers() []*Server{
       servers:= make([]*Server,0)
	   sp.servers.Range(func(key, value interface{}) bool {
		   servers = append(servers, value.(*Server))
		   return true
	   })
	   return servers
}


func LeastConnections(servers []*Server) *Server {
	var bestServer *Server
	for _, server := range servers {
		if bestServer == nil || server.connections < bestServer.connections {
			bestServer = server
		}
	}
	return bestServer

}

func main() {

}