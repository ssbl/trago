package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ssbl/trago/db"
)

const (
	SERVFLAG = "-s"
	SERVCMD = "trago -s {dir}"
	serverUsage = "Run in server mode in the specified directory.\n"
)

var (
	server string
	serverDir string
	clientDir string
	flagDir string
	defaultDir = "(nil)"
)

func init() {
	flag.StringVar(&flagDir, "server", defaultDir, serverUsage)
	flag.StringVar(&flagDir, "s", defaultDir, "Shorthand for --server.")

	log.SetFlags(0)
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if flagDir == defaultDir {
		server, serverDir, clientDir := parseArgs()
		fmt.Printf("%s:%s %s\n", server, serverDir, clientDir)

		err := os.Chdir(clientDir)
		assert(err, "Error changing to directory: %s\n", err)

		cmd := exec.Command("ssh", server,
			strings.Replace(SERVCMD, "{dir}", serverDir, 1))

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		err = cmd.Run()
		assert(err, string(stderr.Bytes()))

		parsed, err := db.Parse(string(stdout.Bytes()))
		assert(err, "Error parsing server response")

		fmt.Println(parsed)
	} else {	  // running in server mode, so we ignore all other flags
		if err := os.Chdir(flagDir); err != nil {
			log.Fatalf("Error changing to directory: %s\n", err)
		}
		
		tradb, err := db.ParseFile()
		assert(err, "Error parsing db file: %s\n", err)

		err = tradb.Update()
		assert(err, "Error updating db file: %s\n", err)

		err = tradb.Write()
		assert(err, "Error writing to db file: %s\n", err)

		bs, err := ioutil.ReadFile(db.TRADB)
		assert(err, "Error reading file: %s\n", err)

		fmt.Println(string(bs))		// send db to stdout
	}
}

func usage() {
	log.Printf("Usage: trago server:dir client-dir\n\n")

	log.Printf("-s <dir>, --server <dir>\n    %s\n", serverUsage);
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

func assert(err error, format string, args ...interface{}) {
	if err != nil {
		log.Fatalf(format, args)
	}
}
