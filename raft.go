package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func main() {
	fmt.Println("vim-go")
}

type State int

const (
	Follower State = iota
	Candidate
	Leader
	Dead
)

type Logs struct {
	Command interface{}
	Term    int
}

type Consesus struct {
	id                 int   // server id
	peers              []int //peers
	server             *Server
	mutex              sync.Mutex
	currentTerm        int
	currentState       State
	electionResetEvent time.Time
	votedFor           int
	logs               []Logs
}

func (cm *Consesus) runElectionTimer() {
	timeout := cm.electionTimeout()
	cm.mutex.Lock()
	term := cm.currentTerm
	cm.mutex.Unlock()
	log.Printf("election timer started %v term %v\n", timeout, term)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		cm.mutex.Lock()
		if cm.cantPing() {
			log.Println("Node is follower")
			cm.mutex.Unlock()
			return
		}
		if term != cm.currentTerm {
			log.Printf("In election timeout term was changed from %d to %d\n", term, cm.currentTerm)
			cm.mutex.Unlock()
			return
		}
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeout {
			cm.startElection()
			cm.mutex.Unlock()
			return
		}
		cm.mutex.Unlock()
	}
}

//expects cm to be locked
func (cm *Consesus) startElection() {
	cm.currentState = Candidate
	cm.currentTerm += 1
	term := cm.currentTerm
	cm.electionResetEvent = time.Now()
	cm.votedFor = cm.id
	log.Printf("Node %d became Candidate In term %d, logs %v\n", cm.id, term, cm.logs)
	cm.ping(term)
}

type RequestVotesArgs struct {
	Term        int
	CandidateId int
}

type VoteReply struct {
	Term  int
	Voted bool
}

func (cm *Consesus) ping(term int) {
	for _, peerId := range cm.peers {
		go func(peerId int) {
			args := RequestVotesArgs{
				Term:        term,
				CandidateId: cm.id,
			}
			log.Printf("Args %+v", args)
			//TODO:: Finish ping
		}(peerId)
	}
}

func (cm *Consesus) cantPing() bool {
	return cm.currentState != Candidate && cm.currentState != Follower
}

func (cm *Consesus) electionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

type Server struct {
}
