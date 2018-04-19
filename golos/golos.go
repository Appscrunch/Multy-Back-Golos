/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package golos

import (
	"log"

	client "github.com/asuleymanov/golos-go"
	pb "github.com/Appscrunch/Multy-Back-Golos/proto"
	"time"
)

// Server is a struct for interaction with golos chain
type Server struct {
	client           *client.Client
	account          string
	activeKey        string
	TrackedAddresses map[string]bool
	BalanceChangedCh chan *pb.Balance
	NewBlockCh chan *pb.Block
}

// NewServer initializes and validates new api struct
// and runs chain monitoring loop
func NewServer(endpoints []string, net, account, key string) (*Server, error) {
	cli, err := client.NewClient(endpoints, net)
	log.Println("new client")
	if err != nil {
		return nil, err
	}
	s := &Server{
		client:           cli,
		account:          account,
		activeKey:        key,
		TrackedAddresses: make(map[string]bool),
	}

	client.Key_List[account] = client.Keys{
		AKey: key,
	}

	s.BalanceChangedCh = make(chan *pb.Balance)
	s.NewBlockCh = make(chan *pb.Block)

	go s.ProcessLoop(0)

	return s, nil
}

// NewBlockLoop checks for new blocks and send them to chans
// start is a number of starting block for iteration
// if start is 0, using head_block_number
func (s *Server) ProcessLoop(start uint32) {
	blockNum := start

	config, err := s.client.Database.GetConfig()
	if err != nil {
		log.Printf("get config: %s", err)
		return
	}

	for {
		props, err := s.client.Database.GetDynamicGlobalProperties()
		if err != nil {
			log.Printf("get global properties: %s", err)
			time.Sleep(time.Duration(config.SteemitBlockInterval) * time.Second)
			continue
		}
		if blockNum == 0 {
			blockNum = props.HeadBlockNumber
		}
		// maybe LastIrreversibleBlockNum, cause possible microforks
		if props.HeadBlockNumber-blockNum > 0 {
			log.Printf("new block: %d", blockNum + 1)
			block, err := s.client.Database.GetBlock(blockNum + 1)
			if err != nil {
				log.Printf("get block: %s", err)
				time.Sleep(time.Duration(config.SteemitBlockInterval) * time.Second)
				continue
			}
			msg := makeBlock(block)
			select {
			case s.NewBlockCh <- &msg:
				// process block, now its only balance change check
				go s.processBalance(block)
			}
			blockNum++
		} else {
			time.Sleep(time.Duration(config.SteemitBlockInterval) * time.Second)
		}
	}
}



