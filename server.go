/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/Appscrunch/Multy-Back-Golos/golos"
	pb "github.com/Appscrunch/Multy-Back-Golos/proto"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

var (
	commit    string
	branch    string
	buildtime string
)

const (
	VERSION = "v0.2"
)

type Server struct {
	*golos.Server
}

// NewServer constructs new server handler
func NewServer(endpoints []string, net, account, key string) (*Server, error) {
	a, err := golos.NewServer(endpoints, net, account, key)
	return &Server{a}, err
}

func run(c *cli.Context) error {
	// check net arguement
	network := c.String("net")
	if network != "test" && network != "golos" {
		return cli.NewExitError(fmt.Sprintf("net must be \"golos\" or \"test\": %s", network), 1)
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

	addr := fmt.Sprintf("%s:%s", c.String("host"), c.String("port"))

	// init gRPC server
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("failed to listen: %s", err), 2)
	}
	// Creates a new gRPC server
	s := grpc.NewServer()
	pb.RegisterNodeCommunicationsServer(s, server)
	log.Printf("listening on %s", addr)
	return cli.NewExitError(s.Serve(lis), 3)
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
