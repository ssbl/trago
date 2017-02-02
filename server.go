package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/ssbl/trago/db"
)

const (
	SERVFLAG = "-s"
	SERVCMD  = "go run $GOPATH/src/github.com/ssbl/trago/server.go -s"
)

var (
	isServer bool
)

type TraServ struct {
	Database *db.TraDb
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
}

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
			} else if msg == "parse\n" {
				_, err := db.ParseFile()
				if err != nil {
					log.Fatal(err)
				}

				bs, err := ioutil.ReadFile(db.TRADB)
				if err != nil {
					log.Fatalf("error reading file: %v\n", err)
				} else {
					fmt.Print(string(bs))
				}
			} else {
				log.Print("got message: " + msg)
			}
		}
	} else {
		cmd := exec.Command("ssh", "localhost", SERVCMD)
		stdout := new(bytes.Buffer)
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatalf("error executing command: %v", err)
		}

		log.Println("waiting on input...")
		for {
			if stdout.Len() != 0 {
				tradb, err := db.Parse(stdout.String())
				if err != nil {
					fmt.Println("error parsing file")
					break
				}
				fmt.Println(tradb)
				stdout.Reset()
				continue
			}
			msg, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
			if err == io.EOF {
				stdin.Close()
				break
			}

			if _, err := stdin.Write(msg); err != nil {
				log.Fatal("error writing to pipe")
			}
		}
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
