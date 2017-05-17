package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/ssbl/trago/rpcdb"
)

const (
	// Test directories, addresses
	LOCALDIR  = "../a"
	REMOTEDIR = "../b"
	LOCALSRV  = "localhost:8999"
	REMOTESRV = "localhost:8998"
	PORT      = ":8999"
)

func init() {
	flag.Usage = usage
}

func main() {
	flag.Parse()

	remoteServ, remoteDir, localDir := parseArgs()
	log.Printf("%s:%s %s\n", remoteServ, remoteDir, localDir)

	// TODO: Is this correct?
	go rpcdb.StartSrv(PORT)

	// TODO: Start the remote trasrv

	if err := rpcdb.Run(localDir, LOCALSRV, REMOTEDIR, REMOTESRV); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	log.Printf("Usage: trago <server>:<dir> <client-dir>\n\n")
}

func parseArgs() (string, string, string) {
	var s, sd, cd string

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	remote := strings.Split(flag.Arg(0), ":")
	if len(remote) != 2 {
		flag.Usage()
		os.Exit(1)
	} else {
		s = strings.TrimSpace(remote[0])
		sd = strings.TrimSpace(remote[1])
	}

	cd = strings.TrimSpace(flag.Arg(1))

	return s, sd, cd
}
