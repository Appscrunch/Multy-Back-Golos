/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package golos

import (
	"log"

	"github.com/asuleymanov/golos-go/types"
	"github.com/asuleymanov/golos-go/apis/database"

	pb "github.com/Appscrunch/Multy-Back-Golos/proto"
	"context"
)

// getNames gets account names of balance changing operations
// based on https://developers.golos.io/golos-v0.17.0/da/d67/structgolos_1_1protocol_1_1base__operation.html
func getNames(rawOp types.Operation) []string {
	switch op := rawOp.Data().(type) {
	case *types.VoteOperation: // vote_operation
		return []string{op.Voter, op.Author}
	case *types.TransferOperation: // transfer_operation
		return []string{op.From, op.To}
	case *types.TransferToVestingOperation: // transfer_to_vesting_operation
		return []string{op.From, op.To}
	case *types.WithdrawVestingOperation: // withdraw_vesting_operation
		return []string{op.Account}
	case *types.LimitOrderCreateOperation: // limit_order_create_operation
		return []string{op.Owner}
	case *types.LimitOrderCancelOperation: // limit_order_cancel_operation
		return []string{op.Owner}
	case *types.ConvertOperation: // convert_operation
		return []string{op.Owner}
	case *types.AccountCreateOperation: // account_create_operation
		return []string{op.Creator}
	case *types.WitnessUpdateOperation: // witness_update_operation
		return []string{op.Owner}
	case *types.POWOperation: // pow_operation
		return []string{op.WorkerAccount}
	case *types.SetWithdrawVestingRouteOperation: // set_withdraw_vesting_route_operation
		return []string{op.FromAccount, op.ToAccount}
	case *types.LimitOrderCreate2Operation: // limit_order_create2_operation
		return []string{op.Qwner} // TODO: fix typo in golos-go
	case *types.EscrowTransferOperation: // escrow_transfer_operation
		return []string{op.From}
	case *types.EscrowReleaseOperation: // escrow_release_operation
		return []string{op.From, op.To, op.Agent}
	case *types.POW2Operation: // pow2_operation
		if op.Input != nil {
			return []string{op.Input.WorkerAccount}
		}
		return []string{}
	case *types.TransferToSavingsOperation: // transfer_to_savings_operation
		return []string{op.From, op.To}
	case *types.TransferFromSavingsOperation: // transfer_from_savings_operation
		return []string{op.From, op.To}
	case *types.ClaimRewardBalanceOperation: // claim_reward_balance_operation
		return []string{op.Account}
	case *types.DelegateVestingSharesOperation: // delegate_vesting_shares_operation
		return []string{op.Delegatee, op.Delegator}
	case *types.AccountCreateWithDelegationOperation: // account_create_with_delegation_operation
		return []string{op.Creator, op.NewAccountName}
	case *types.FillConvertRequestOperation: // fill_convert_request_operation
		return []string{op.Owner}
	case *types.AuthorRewardOperation: // author_reward_operation
		return []string{op.Author}
	case *types.CurationRewardOperation: // curation_reward_operation
		return []string{op.CommentAuthor, op.Curator}
	case *types.CommentRewardOperation: // comment_reward_operation
		return []string{op.Author}
	case *types.LiquidityRewardOperation: // liquidity_reward_operation
		return []string{op.Owner}
	case *types.InterestOperation: // interest_operation
		return []string{op.Owner}
	case *types.FillVestingWithdrawOperation: // fill_vesting_withdraw_operation
		return []string{op.FromAccount, op.ToAccount}
	case *types.FillOrderOperation: // fill_order_operation
		return []string{op.CurrentOwner, op.OpenOwner}
	case *types.FillTransferFromSavingsOperation: // fill_transfer_from_savings_operation
		return []string{op.From, op.To}
	case *types.ReturnVestingDelegationOperation: // return_vesting_delegation_operation
		return []string{op.Account}
	case *types.CommentBenefactorRewardOperation: // comment_benefactor_reward_operation
		return []string{op.Author, op.Benefactor}
	}
	return nil
}

// processBalance finds ops that changes balance that involves tracked addresses
// and pushes updated balances to chanel
func (s *Server) processBalance(block *database.Block) {
	changedBalance := map[string]bool{}
	checkAddrs := []string{}
	addrs := []string{}
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			addrs = append(addrs, getNames(op)...)
		}
	}
	for _, addr := range addrs {
		// if already in checking
		if _, ok := changedBalance[addr]; !ok {
			// if tracked
			if _, ok := s.TrackedAddresses[addr]; ok {
				changedBalance[addr] = true
				checkAddrs = append(checkAddrs, addr)
			}
		}
	}
	if len(checkAddrs) > 0 {
		log.Printf("balances changed %v", checkAddrs)
		balances, err := s.EventGetBalances(context.Background(), &pb.Accounts{Names:checkAddrs})
		if err != nil {
			// BUG: unchecked balances for block on error
			log.Printf("get balance: %s", err)
			return
		}
		for _, b := range balances.Balances {
			s.BalanceChangedCh <- b
		}
	}
	return
}

// makeBlock converts golos-go Block type to protobuf-specified Block type
func makeBlock(block *database.Block) (pb.Block) {
	pbBlock := pb.Block{
		Height:block.Number,
		Time:block.Timestamp.Unix(),
		Transactions: make([]*pb.Block_Transaction, len(block.Transactions)),
	}
	for i, tx := range block.Transactions {
		pbTx := pb.Block_Transaction{
			RefBlockNum: uint32(tx.RefBlockNum),
			RefBlockPrefix:uint32(tx.RefBlockPrefix),
			Expiration: tx.Expiration.Unix(),
			Signatures: tx.Signatures,
		}

		ops, err := tx.Operations.MarshalJSON()
		if err != nil {
			log.Printf("Error marshaling operations: %s", err)
		}

		pbTx.Operations = string(ops)

		pbBlock.Transactions[i] = &pbTx
	}
	return pbBlock
}