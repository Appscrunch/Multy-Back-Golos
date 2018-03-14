package api

import (
	"encoding/json"

	"github.com/asuleymanov/golos-go/types"
)

// OkErrResponse is a basic completion response with optional error
type OkErrResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type AccountCheckRequest struct {
	Name string `json:"name"`
}

type AccountCheckResponse struct {
	Exist bool   `json:"exist"`
	Error string `json:"error"`
}

type GetBalancesRequest struct {
	Accounts []string `json:"accounts"`
}

type GetBalancesResponse struct {
	Balances []*Balance `json:"balances"`
	Error    string     `json:"error"`
}

type AccountCreateRequest struct {
	Account string `json:"account"`
	Owner   string `json:"owner"`
	Active  string `json:"active"`
	Posting string `json:"posting"`
	Memo    string `json:"memo"`
}

type AccountCreateResponse = OkErrResponse

type BalancesChangedMessage struct {
	Balances []*Balance `json:"balances"`
}

type NewBlockMessage struct {
	Height uint32 `json:"height,omitempty"`
	// Hash       string   `json:"hash,omitempty"`
	Time         int64                `json:"time,omitempty"`
	Transactions []*types.Transaction `json:"transactions"`
}

type TrackAddressesRequest struct {
	Adresses []string `json:"adresses,omitempty"`
}

type TrackAddressesResponse = OkErrResponse

type GetTrackedAddressesRequest struct {
}

type GetTrackedAddressesResponse struct {
	Accounts []string `json:"accounts"`
	Error    string   `json:"error,omitempty"`
}

type SendTransactionRequest = types.Transaction

type SendTransactionResponse struct {
	Ok       bool             `json:"ok"`
	Error    string           `json:"error,omitempty"`
	Response *json.RawMessage `json:"response,omitempty"`
}
