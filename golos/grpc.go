/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package golos

import (
	"context"

	"github.com/asuleymanov/golos-go/types"

	pb "github.com/Appscrunch/Multy-Back-Golos/proto"
	"log"
	"encoding/json"
)

// EventGetBlockInfo returns head block info
func (s *Server) EventGetHeadInfo(ctx context.Context, _ *pb.Empty) (*pb.HeadInfo, error){
	props, err := s.client.Database.GetDynamicGlobalProperties()
	if err != nil {
		return nil, err
	}
	resp := &pb.HeadInfo{
		Height:props.HeadBlockNumber,
		Id:props.HeadBlockID,
	}
	return resp, nil
}

// EventAccountCheck checks if account already exists
// returns true if account exists
func (s *Server) EventAccountCheck(ctx context.Context, request *pb.AccountCheckRequest) (*pb.AccountCheckResponse, error) {
	accs, err := s.client.Database.GetAccounts([]string{request.Name})
	if err != nil {
		return &pb.AccountCheckResponse{
			Exist:true,
			Error: err.Error(),
		}, err
	}
	if len(accs) == 0 {
		return &pb.AccountCheckResponse{
			Exist: false,
		}, nil
	} else {
		return &pb.AccountCheckResponse{
			Exist:true,
		}, nil
	}
}

// EventAccountCreate creates account by constructing
// account_create operation and broadcasting it
// account is account name
// fee is account creation fee if "0.000 GOLOS" format
// owner, active, posting, memo is a public keys
func (s *Server) EventAccountCreate(ctx context.Context, request *pb.AccountCreateRequest) (*pb.OkErrResponse, error) {
	//account, fee, owner, active, posting, memo string
	var ops []types.Operation

	// construct operation
	op := &types.AccountCreateOperation{
		Fee:            request.Fee,
		Creator:        s.account,
		NewAccountName: request.Account,
		Owner:          authorityFromKey(request.Owner),
		Active:         authorityFromKey(request.Active),
		Posting:        authorityFromKey(request.Posting),
		MemoKey:        request.Memo,
		JsonMetadata:   "{}",
	}

	ops = append(ops, op)

	resp, err := s.client.SendTrx(s.account, ops)

	if err != nil {
		log.Printf("AccountCreate error: resp: %s, req: %s", resp, request)
	}

	return &pb.OkErrResponse{
		Ok: err == nil,
		Error: err.Error(),
	}, err
}

// authorityFromKey contructs golos-go/types Authority struct from public key
// using https://developers.golos.io/golos-v0.17.0/dc/d58/structgolos_1_1protocol_1_1authority.html
// and https://steemit.com/dsteem/@andravasko/how-to-creating-an-account-with-dsteem-0-6-2017928t16287166z
// cause weights are confusing
func authorityFromKey(key string) *types.Authority {
	return &types.Authority{
		WeightThreshold: 1,
		KeyAuths: types.StringInt64Map{
			key: 1,
		},
		AccountAuths: types.StringInt64Map{},
	}
}


// GetBalances gets balances of multiple accounts at once
// using get_accounts rpc call
func (s *Server) EventGetBalances(ctx context.Context, request *pb.Accounts) (*pb.GetBalancesResponse, error) {
	accs, err := s.client.Database.GetAccounts(request.Names)
	if err != nil {
		return nil, err
	}
	balances := make([]*pb.Balance, len(accs))
	for i, acc := range accs {
		balances[i] = &pb.Balance{
			Name:              acc.Name,
			Balance:           acc.Balance,
			SavingsBalance:    acc.SavingsBalance,
			SbdBalance:        acc.SbdBalance,
			SavingsSbdBalance: acc.SavingsSbdBalance,
			VestingBalance:    acc.VestingBalance,
		}
	}
	return &pb.GetBalancesResponse{
		Balances: balances,
	}, nil
}

// TrackAddresses adds addresses for tracking
func (s *Server) EventTrackAddresses(ctx context.Context, request *pb.Accounts) (*pb.OkErrResponse, error) {
	for _, addr := range request.Names {
		s.TrackedAddresses[addr] = true
	}
	return &pb.OkErrResponse{
		Ok: true,
	}, nil
}

// GetTrackedAddresses gets currently tracked accounts names
func (s *Server) EventGetTrackedAddresses(ctx context.Context, _ *pb.Empty) (*pb.Accounts, error) {
	accounts := make([]string, 0, len(s.TrackedAddresses))
	for k := range s.TrackedAddresses {
		accounts = append(accounts, k)
	}
	return &pb.Accounts{
		Names:accounts,
	}, nil

}

// SendTransaction syncronously broadcasts constructed transaction to a chain
func (s *Server) EventSendTransactionJSON(ctx context.Context, trxRaw *pb.TransactionJSON) (*pb.SendTransactionResponse, error) {
	trx := types.Transaction{}
	err := json.Unmarshal([]byte(trxRaw.Json), &trx)
	if err != nil {
		return nil, err
	}
	bResp, err := s.client.NetworkBroadcast.BroadcastTransactionSynchronousRaw(&trx)
	return &pb.SendTransactionResponse{
		Error: err.Error(),
		Ok: err == nil,
		Response: string(*bResp),
	}, nil
}

func (s *Server) BalanceChanged(_ *pb.Empty, stream pb.NodeCommunications_BalanceChangedServer) error {
	for {
		select {
		case balance := <- s.BalanceChangedCh:
			log.Printf("Balance changed %v", balance.String())
			stream.Send(balance)
			
		}
	}
	return nil
}

func (s *Server) NewBlock(_ *pb.Empty, stream pb.NodeCommunications_NewBlockServer) error {
	for {
		select {
		case block := <- s.NewBlockCh:
			log.Printf("Balance changed %v", block.String())
			stream.Send(block)
		}
	}
	return nil
}