package api

import (
	"encoding/json"
	"log"
	"time"

	client "github.com/asuleymanov/golos-go"
	"github.com/asuleymanov/golos-go/apis/database"
	"github.com/asuleymanov/golos-go/types"
)

// API is a struct for interaction with golos chain
type API struct {
	client           *client.Client
	account          string
	activeKey        string
	TrackedAddresses map[string]bool
}

// NewAPI initializes and validates new api struct
func NewAPI(endpoints []string, net, account, key string) (*API, error) {
	cli, err := client.NewClient(endpoints, net)
	log.Println("new client")
	if err != nil {
		return nil, err
	}
	api := &API{
		client:           cli,
		account:          account,
		activeKey:        key,
		TrackedAddresses: make(map[string]bool),
	}

	client.Key_List[account] = client.Keys{
		AKey: key,
	}
	return api, nil
}

// Balance is a struct of all available balances
// basically it's a balance subset of database.Account struct
type Balance struct {
	Name              string `json:"name"`
	Balance           string `json:"balance"`
	SavingsBalance    string `json:"savings_balance"`
	SbdBalance        string `json:"sbd_balance"`
	SavingsSbdBalance string `json:"savings_sbd_balance"`
	VestingBalance    string `json:"vesting_balance"`
}

// GetBalances gets balances of multiple accounts at once
// using get_accounts rpc call
// accounts is a slice of account names
func (api *API) GetBalances(accounts []string) ([]*Balance, error) {
	accs, err := api.client.Database.GetAccounts(accounts)
	if err != nil {
		return nil, err
	}
	balances := make([]*Balance, len(accs))
	for i, acc := range accs {
		balances[i] = &Balance{
			Name:              acc.Name,
			Balance:           acc.Balance,
			SavingsBalance:    acc.SavingsBalance,
			SbdBalance:        acc.SbdBalance,
			SavingsSbdBalance: acc.SavingsSbdBalance,
			VestingBalance:    acc.VestingBalance,
		}
	}
	return balances, nil
}

// AccountCheck checks if account already exists
// returns true if account exists
func (api *API) AccountCheck(account string) (exists bool, err error) {
	accs, err := api.client.Database.GetAccounts([]string{account})
	if err != nil {
		return true, err
	}
	if len(accs) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

// GetBalance fetches balances of single account
// using GetBalances
func (api *API) GetBalance(account string) (*Balance, error) {
	balances, err := api.GetBalances([]string{account})
	if err != nil {
		return nil, err
	}
	return balances[0], nil
}

// GetConfig gets node config
func (api *API) GetConfig() (*database.Config, error) {
	return api.client.Database.GetConfig()
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

// AccountCreate creates account by constructing
// account_create operation and broadcasting it
// account is account names
// fee is account creation fee if "0.000 GOLOS" format
// owner, active, posting, memo is a public keys
func (api *API) AccountCreate(account, fee, owner, active, posting, memo string) error {
	var ops []types.Operation

	// construct operation
	op := &types.AccountCreateOperation{
		Fee:            fee,
		Creator:        api.account,
		NewAccountName: account,
		Owner:          authorityFromKey(owner),
		Active:         authorityFromKey(active),
		Posting:        authorityFromKey(posting),
		MemoKey:        memo,
		JsonMetadata:   "{}",
	}

	ops = append(ops, op)

	resp, err := api.client.SendTrx(api.account, ops)

	log.Printf("Response: %s", resp)

	return err
}

// TrackAddresses adds addresses for tracking
// addresses is a slice of account names
func (api *API) TrackAddresses(addresses []string) error {
	for _, addr := range addresses {
		api.TrackedAddresses[addr] = true
	}
	return nil
}

// GetTrackedAddresses gets currently tracked accounts names
func (api *API) GetTrackedAddresses() ([]string, error) {
	accounts := make([]string, 0, len(api.TrackedAddresses))
	for k, _ := range api.TrackedAddresses {
		accounts = append(accounts, k)
	}
	return accounts, nil

}

// SendTransaction syncronously broadcasts constructed transaction to a chain
func (api *API) SendTransaction(trx *types.Transaction) (*json.RawMessage, error) {
	return api.client.NetworkBroadcast.BroadcastTransactionSynchronousRaw(trx)
}

// NewBlockLoop checks for new blocks and send them to chans
// start is a number of starting block for iteration
// if start is 0, using head_block_number
func (api *API) NewBlockLoop(blockChan chan<- *NewBlockMessage, balanceChan chan<- *BalancesChangedMessage, done <-chan bool, start uint32) {
	blockNum := start

	config, err := api.client.Database.GetConfig()
	if err != nil {
		log.Printf("get config: %s", err)
		return
	}

	for {
		props, err := api.client.Database.GetDynamicGlobalProperties()
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
			block, err := api.client.Database.GetBlock(blockNum + 1)
			if err != nil {
				log.Printf("get block: %s", err)
				time.Sleep(time.Duration(config.SteemitBlockInterval) * time.Second)
				continue
			}
			msg := &NewBlockMessage{
				Height:       block.Number,
				Time:         block.Timestamp.Unix(),
				Transactions: block.Transactions,
			}
			select {
			case <-done:
				close(blockChan)
				close(balanceChan)
				log.Println("end new block loop")
				return
			case blockChan <- msg:
				// process block, now its only balance change check
				go api.processBalance(block, balanceChan, done)
			}
			blockNum++
		} else {
			time.Sleep(time.Duration(config.SteemitBlockInterval) * time.Second)
		}
	}
}

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
func (api *API) processBalance(block *database.Block, balanceChan chan<- *BalancesChangedMessage, done <-chan bool) {
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
			if _, ok := api.TrackedAddresses[addr]; ok {
				changedBalance[addr] = true
				checkAddrs = append(checkAddrs, addr)
			}
		}
	}
	if len(checkAddrs) > 0 {
		balances, err := api.GetBalances(checkAddrs)
		if err != nil {
			// BUG: unchecked balances for block on error
			log.Printf("get balance: %s", err)
			return
		}
		msg := &BalancesChangedMessage{
			Balances: balances,
		}
		select {
		case <-done:
			log.Println("process block done")
			return
		case balanceChan <- msg:
			return
		}
	}
	return
}
