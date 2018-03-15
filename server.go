package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Appscrunch/Multy-Back-Golos/api"

	socketio "github.com/googollee/go-socket.io"
	"github.com/urfave/cli"
)

var (
	commit    string
	branch    string
	buildtime string
)

const (
	VERSION = "v0.1"

	ROOM                        = "golos"
	EVENT_CONNECTION            = "connection"
	EVENT_CREATE_ACCOUNT        = "account:create"
	EVENT_CHECK_ACCOUNT         = "account:check"
	EVENT_BALANCE_GET           = "balance:get"
	EVENT_BALANCE_CHANGED       = "balance:changed"
	EVENT_TRACK_ADDRESSES       = "balance:track:add"
	EVENT_GET_TRACKED_ADDRESSES = "balance:track:get"
	EVENT_SEND_TRANSACTION      = "transaction:send"
	EVENT_NEW_BLOCK             = "block:new"
)

type Server struct {
	*api.API
}

// stringify marshals data struct to a string
// it ignores marshal data, so use it on knownly valid structs
func stringify(data interface{}) string {
	s, err := json.Marshal(data)
	if err != nil {
		log.Printf("stringify: %s", err)
	}
	return string(s)
}

// errStr converts nil errors to an empty string
// and non-nil errors to it string representation
func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// NewServer constructs new server handler
func NewServer(endpoints []string, net, account, key string) (*Server, error) {
	a, err := api.NewAPI(endpoints, net, account, key)
	return &Server{a}, err
}

func unmarshalRequest(data interface{}, v interface{}) error {
	return json.Unmarshal([]byte(stringify(data)), v)
}

func (s *Server) onAccountCheck(data interface{}) string {
	req := api.AccountCheckRequest{}
	err := unmarshalRequest(data, req)
	if err != nil {
		return stringify(api.OkErrResponse{
			// account exists by default, so you cannot register one on error
			Ok:    true,
			Error: errStr(err),
		})
	}

	exist, err := s.AccountCheck(req.Name)
	return stringify(api.OkErrResponse{
		Ok:    exist,
		Error: errStr(err),
	})
}

func (s *Server) onAccountCreate(data interface{}) string {
	req := api.AccountCreateRequest{}
	err := json.Unmarshal(data.([]byte), &req)
	if err != nil {
		stringify(api.OkErrResponse{
			Ok:    false,
			Error: errStr(err),
		})
	}
	err = s.AccountCreate(req.Account, "", req.Owner, req.Active, req.Posting, req.Memo)
	return stringify(api.OkErrResponse{
		Ok:    err == nil,
		Error: errStr(err),
	})

}

func (s *Server) onGetBalances(data interface{}) string {
	req := api.GetBalancesRequest{}
	err := unmarshalRequest(data, &req)
	if err != nil {
		resp, _ := json.Marshal(api.GetBalancesResponse{
			Balances: nil,
			Error:    errStr(err),
		})
		return string(resp)
	}
	balances, err := s.GetBalances(req.Accounts)
	return stringify(api.GetBalancesResponse{
		Balances: balances,
		Error:    errStr(err),
	})
}

func (s *Server) onTrackAddresses(data interface{}) string {
	req := api.TrackAddressesRequest{}
	err := unmarshalRequest(data, &req)
	if err != nil {
		return stringify(api.TrackAddressesResponse{
			Ok:    false,
			Error: errStr(err),
		})
	}
	err = s.TrackAddresses(req.Adresses)
	return stringify(api.TrackAddressesResponse{
		Ok:    err == nil,
		Error: errStr(err),
	})
}

func (s *Server) onGetTrackedAddresses(data interface{}) string {
	// req := api.GetTrackedAddressesRequest{}
	// err := unmarshalRequest(data, &req)
	// if err != nil {
	// 	return stringify(api.GetTrackedAddressesResponse{
	// 		Accounts: nil,
	// 		Error:    errStr(err),
	// 	})
	// }
	accs, err := s.GetTrackedAddresses()
	return stringify(api.GetTrackedAddressesResponse{
		Accounts: accs,
		Error:    errStr(err),
	})
}

func (s *Server) onSendTransaction(data interface{}) string {
	req := api.SendTransactionRequest{}
	err := unmarshalRequest(data, &req)
	if err != nil {
		return stringify(api.SendTransactionResponse{
			Ok:    false,
			Error: errStr(err),
		})
	}
	bResp, err := s.SendTransaction(&req)
	return stringify(api.SendTransactionResponse{
		Ok:       err == nil,
		Error:    errStr(err),
		Response: bResp,
	})
}

func (s *Server) broadcastLoop(socket *socketio.Server) {
	blockChan := make(chan *api.NewBlockMessage)
	balanceChan := make(chan *api.BalancesChangedMessage)
	done := make(chan bool)
	go s.NewBlockLoop(blockChan, balanceChan, done, 0)
	for {
		select {
		case block := <-blockChan:
			log.Printf("broadcast new block: %s", stringify(block))
			socket.BroadcastTo(ROOM, EVENT_NEW_BLOCK, stringify(block))
		case balances := <-balanceChan:
			log.Println("broadcast balance change")
			socket.BroadcastTo(ROOM, EVENT_BALANCE_CHANGED, stringify(balances))
		}
	}
	// TODO: graceful shutdown
}

func run(c *cli.Context) error {
	// check net arguement
	net := c.String("net")
	if net != "test" && net != "golos" {
		return cli.NewExitError(fmt.Sprintf("net must be \"golos\" or \"test\": %s", net), 1)
	}

	server, err := NewServer(
		[]string{c.String("node")},
		c.String("net"),
		c.String("account"),
		c.String("key"),
	)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("cannot init server: %s", err), 2)
	}
	log.Println("new server")

	socket, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	socket.On(EVENT_CONNECTION, func(so socketio.Socket) {
		so.Join(ROOM)
		so.On(EVENT_CHECK_ACCOUNT, server.onAccountCheck)
		so.On(EVENT_CREATE_ACCOUNT, server.onAccountCreate)
		so.On(EVENT_BALANCE_GET, server.onGetBalances)
		so.On(EVENT_TRACK_ADDRESSES, server.onTrackAddresses)
		so.On(EVENT_GET_TRACKED_ADDRESSES, server.onGetTrackedAddresses)
		so.On(EVENT_SEND_TRANSACTION, server.onSendTransaction)

	})

	http.Handle("/socket.io/", socket)
	hostport := fmt.Sprintf("%s:%s", c.String("host"), c.String("port"))
	log.Println("Serving at", hostport)
	go server.broadcastLoop(socket)
	return cli.NewExitError(http.ListenAndServe(hostport, nil), 3)
}

func main() {
	app := cli.NewApp()
	app.Name = "multy-golos"
	app.Usage = `Golos node socket.io API for Multy backend`
	app.Version = fmt.Sprintf("%s (commit: %s, branch: %s, buildtime: %s)", VERSION, commit, branch, buildtime)
	app.Author = "vovapi"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host",
			Usage:  "hostname to bind to",
			EnvVar: "MULTY_GOLOS_HOST",
			Value:  "",
		},
		cli.StringFlag{
			Name:   "port",
			Usage:  "port to bind to",
			EnvVar: "MULTY_GOLOS_PORT",
			Value:  "8080",
		},
		cli.StringFlag{
			Name:   "node",
			Usage:  "node websocker address",
			EnvVar: "MULTY_GOLOS_NODE",
		},
		cli.StringFlag{
			Name:   "net",
			Usage:  `network: "golos" for mainnet or "test" for testnet`,
			EnvVar: "MULTY_GOLOS_NET",
			Value:  "test",
		},
		cli.StringFlag{
			Name:   "account",
			Usage:  "golos account for user registration",
			EnvVar: "MULTY_GOLOS_ACCOUNT",
		},
		cli.StringFlag{
			Name:   "key",
			Usage:  "active key for specified user for user registration",
			EnvVar: "MULTY_GOLOS_KEY",
		},
	}
	app.Action = run
	app.Run(os.Args)
}
