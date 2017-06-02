package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ssbl/trago/rpcdb"
)

const (
	LOCALSRV = "localhost:8999"
	PORT     = ":8999"
	SRVPORT  = ":8998"
)

var (
	serverMode bool
	sshInit    bool
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

	cmd := exec.Command("ssh", remoteAddr, "trago", "-s")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	remoteServ := remoteAddr + SRVPORT
	if err := rpcdb.Run(localDir, LOCALSRV, remoteDir, remoteServ); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	log.Printf("Usage: trago <remote-address>:<remote-dir> <local-dir>\n")
}

func parseArgs() (string, string, string) {
	var remoteAddr, remoteDir, clientDir string

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	remote := strings.Split(flag.Arg(0), ":")
	if len(remote) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	remoteAddr = strings.TrimSpace(remote[0])
	remoteDir = strings.TrimSpace(remote[1])
	clientDir = strings.TrimSpace(flag.Arg(1))

	return remoteAddr, remoteDir, clientDir
}
