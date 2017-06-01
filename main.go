package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	// "os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ssbl/trago/rpcdb"
)

const (
	LOCALSRV = "localhost:8999"
	PORT     = ":8999"
	SRVPORT  = ":8998"
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

	username, hostname, remoteDir, localDir := parseArgs()

	fmt.Print("SSH password: ")
	password, _ := terminal.ReadPassword(int(syscall.Stdin))
	pass := string(password)

	// TODO: Is this correct?
	go rpcdb.StartSrv(PORT)

	// cmd := exec.Command("ssh", remoteAddr, "trago -s")
	// if err := cmd.Start(); err != nil {
	// 	log.Fatal(err)
	// }
	err := startRemote(username, hostname, pass)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for remote to start.
	select { // TODO: Find a better way to handle this
	case <-time.After(3 * time.Second):
	}

	remoteServ := hostname + SRVPORT
	if err := rpcdb.Run(localDir, LOCALSRV, remoteDir, remoteServ); err != nil {
		log.Fatal(err)
	}
}

func startRemote(user, host, pass string) error {
	// var hostKey ssh.PublicKey

	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":22", &config)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run("/usr/bin/whoami")
	log.Println(b.String())
	return err
}

func usage() {
	log.Printf("Usage: trago <username>@<hostname>:<remote-dir> <local-dir>\n")
}

func parseArgs() (string, string, string, string) {
	var username, hostname, serverDir, clientDir string

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	remote := strings.Split(flag.Arg(0), ":")
	if len(remote) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	serverAddress := strings.Split(strings.TrimSpace(remote[0]), "@")
	if len(serverAddress) != 2 {
		flag.Usage()
		os.Exit(1)
	}
	username = strings.TrimSpace(serverAddress[0])
	hostname = strings.TrimSpace(serverAddress[1])
	serverDir = strings.TrimSpace(remote[1])
	clientDir = strings.TrimSpace(flag.Arg(1))

	return username, hostname, serverDir, clientDir
}
