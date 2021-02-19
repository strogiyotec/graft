package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
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
	// apiUrl: 'http://localhost:8080',
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

type AppendEntriesReply struct {
	TermId  int
	Success bool
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
	Entries  []Logs
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
	cm.ping(term, term)
}

type RequestVotesArgs struct {
	Term        int
	CandidateId int
}

type VoteReply struct {
	Term  int
	Voted bool
}

func (cm *Consesus) ping(term int, currentTerm int) {
	var votesReceived int32 = 1
	for _, peerId := range cm.peers {
		go func(peerId int) {
			args := RequestVotesArgs{
				Term:        term,
				CandidateId: cm.id,
			}
			var reply VoteReply
			log.Printf("Sending request to %d %+v\n", peerId, args)
			if err := cm.server.requestVote(peerId, args, reply); err != nil {
				cm.mutex.Lock()
				defer cm.mutex.Unlock()
				log.Printf("Received reply %+v from peer %d\n", reply, peerId)
				if cm.currentState != Candidate {
					log.Printf("While waiting for reply state changed to %d\n", cm.currentState)
					return
				}
				if reply.Term > currentTerm {
					log.Printf("Term is out of date %d\n", reply.Term)
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == currentTerm {
					if reply.Voted {
						votes := int(atomic.AddInt32(&votesReceived, 1))
						if cm.wonElection(votes) {
							log.Printf("Node %d won election\n", cm.id)
							cm.startLeader()
							return
						}
					}
				}
			}
		}(peerId)
	}
}

//TODO implement
func (cm *Consesus) sendHeartBeats() {
	cm.mutex.Lock()
	savedTerm := cm.currentTerm
	cm.mutex.Unlock()
	for _, peerId := range cm.peers {
		args := AppendEntriesArgs{
			Term:     savedTerm,
			LeaderId: cm.id,
		}
		go func(peerId int) {
			log.Printf("Sending logs to %d\n", peerId)
			var reply AppendEntriesReply
			if err := cm.server.appendEntriesCall(peerId, args, &reply); err == nil {
				cm.mutex.Lock()
				defer cm.mutex.Unlock()
				if reply.TermId > savedTerm {
					log.Println("term out of date in reply")
					cm.becomeFollower(reply.TermId)
					return
				}
			}
		}(peerId)
	}
}

func (cm *Consesus) startLeader() {
	cm.currentState = Leader
	log.Printf("become leader , term %d log=%v\n", cm.currentTerm, cm.logs)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			cm.sendHeartBeats()
			<-ticker.C
			cm.mutex.Lock()
			if cm.currentState != Leader {
				log.Printf("Node %d is not a leader anymore\n", cm.id)
				cm.mutex.Unlock()
				return
			}
			cm.mutex.Unlock()
		}

	}()
}

func (cm *Consesus) wonElection(votes int) bool {
	return votes*2 > len(cm.peers)+1
}

func (cm *Consesus) becomeFollower(term int) {
	log.Printf("Become follower with term %d\n", term)
	cm.currentState = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()
	go cm.runElectionTimer()
}

func (cm *Consesus) cantPing() bool {
	return cm.currentState != Candidate && cm.currentState != Follower
}

func (cm *Consesus) electionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
