package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ssbl/trago/rpcdb"
)

const (
	LOCALSRV  = "localhost:8999"
	PORT      = ":8999"
	SRVPORT   = ":8998"
)

var (
	serverMode bool
)

func init() {
	flag.BoolVar(&serverMode, "s", false, "Run in server mode.\n")

	flag.Usage = usage
}

func main() {
	flag.Parse()

	if serverMode {
		rpcdb.StartSrv(SRVPORT)
	}

	remoteAddr, remoteDir, localDir := parseArgs()

	// TODO: Is this correct?
	go rpcdb.StartSrv(PORT)

	cmd := exec.Command("ssh", remoteAddr, "trago -s")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Wait for remote to start.
	select {					// TODO: Find a better way to handle this
		case <- time.After(2 * time.Second):
	}

	remoteServ := remoteAddr + SRVPORT
	if err := rpcdb.Run(localDir, LOCALSRV, remoteDir, remoteServ); err != nil {
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
