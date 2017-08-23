package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ssbl/trago/tra"
)

const (
	PORT     = ":8999"
	SRVPORT  = ":8998"
	LOCALSRV = "localhost" + PORT
)

var (
	serverMode bool
	sshInit    bool
)

func init() {
	flag.BoolVar(&serverMode, "s", false, "Run in server mode.\n")

	log.SetFlags(0)
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if serverMode {
		tra.StartSrv(SRVPORT)
	}

	hostname, remoteAddr, remoteDir, localDir := parseArgs()

	// TODO: Is this correct?
	go tra.StartSrv(PORT)

	cmd := exec.Command("ssh", remoteAddr, "trago", "-s")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	remoteServ := hostname + SRVPORT
	if err := tra.Run(localDir, LOCALSRV, remoteDir, remoteServ); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	log.Printf("Usage: trago <user>@<hostname>:<remote-dir> <local-dir>\n")
}

func parseArgs() (string, string, string, string) {
	var hostname, remoteAddr, remoteDir, clientDir string

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	remote := strings.Split(flag.Arg(0), ":")
	if len(remote) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	host := strings.Split(remote[0], "@")
	if len(host) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	hostname = strings.TrimSpace(host[1])
	remoteAddr = strings.TrimSpace(remote[0])
	remoteDir = strings.TrimSpace(remote[1])
	clientDir = strings.TrimSpace(flag.Arg(1))

	return hostname, remoteAddr, remoteDir, clientDir
}
