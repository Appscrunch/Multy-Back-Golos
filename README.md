# Multy-Back-Golos
Golos node socket.io API for Multy backend

## Installation
First, clone the repo:
```bash
git clone https://github.com/Appscrunch/Multy-Back-golos
cd Multy-Back-Golos
```
Then sync the dependencies using [govendor](https://github.com/kardianos/govendor):
```bash
govendor sync
```
And build:
```bash
go build -o multy-golos
```

## Usage
Check out help (notice optional initialization using environment variables):
```bash
$ ./multy-golos -h
NAME:
   multy-golos - Golos node socket.io API for Multy backend

USAGE:
   multy-golos [global options] command [command options] [arguments...]

VERSION:
   v0.1

AUTHOR(S):
   vovapi

COMMANDS:
GLOBAL OPTIONS:
   --host 		hostname to bind to [$MULTY_GOLOS_HOST]
   --port "8080"	port to bind to [$MULTY_GOLOS_PORT]
   --node 		node websocker address [$MULTY_GOLOS_NODE]
   --net "test"		network: "golos" for mainnet or "test" for testnet [$MULTY_GOLOS_NET]
   --account 		golos account for user registration [$MULTY_GOLOS_ACCOUNT]
   --key 		active key for specified user for user registration [$MULTY_GOLOS_KEY]
   --help, -h		show help
   --version		print the version
```
## API
Checkout events in [server.go](server.go):
```go
const (
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
```
Checkout messages types in [api/messages.go](api/messages.go)

## TODO
* Dockerfile
* Graceful shutdown
* Check for valid account name in `account:check`
