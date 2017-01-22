package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
)

const (
	SERVFLAG = "-s"
	SERVCMD  = "go run $GOPATH/src/trago/server.go -s"
)

var (
	isServer bool
)

func main() {
	if isElem(os.Args, SERVFLAG) {
		isServer = true
		log.Println("running in server mode...")
	}

	if isServer {
		for {
			msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err == io.EOF {
				log.Println("got EOF, exiting...")
				break
			}

			log.Println("got message: " + msg)
		}
	} else {
		cmd := exec.Command("ssh", "localhost", SERVCMD)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatalf("error executing command: %v", err)
		}

		log.Println("waiting on input...")
		msg, _ := bufio.NewReader(os.Stdin).ReadBytes('\n')

		if _, err := stdin.Write(msg); err != nil {
			log.Fatal("error writing to pipe")
		}

		stdin.Close()			// looks like this sends EOF
	}
}

func isElem(haystack []string, needle string) bool {
	for _, elem := range haystack {
		if elem == needle {
			return true
		}
	}
	return false
}
