package main

import (
	"fmt"
	"net/rpc"
	"sync"
)

const (
	requestVoteCall   string = "request.vote"
	appendEntriesCall string = "request.vote"
)

type Server struct {
	mu      sync.Mutex
	clients map[int]*rpc.Client
}

func (server *Server) appendEntriesCall(peerId int, args interface{}, reply interface{}) error {
	server.mu.Lock()
	peer := server.clients[peerId]
	server.mu.Unlock()
	if peer == nil {
		return fmt.Errorf("Client %d is closed", peerId)
	} else {
		return peer.Call(appendEntriesCall, args, reply)
	}
}
func (server *Server) requestVote(id int, args interface{}, reply interface{}) error {
	server.mu.Lock()
	peer := server.clients[id]
	server.mu.Unlock()
	if peer == nil {
		return fmt.Errorf("Client %d is closed", id)
	} else {
		return peer.Call(requestVoteCall, args, reply)
	}
}
