package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
		log.Printf("%s:%s %s\n", server, serverDir, clientDir)
	} else {	  // running in server mode, so we ignore all other flags
		if err := os.Chdir(flagDir); err != nil {
			log.Fatalf("Error changing to directory: %s\n", err)
		}
		
		tradb, err := db.ParseFile()
		if err != nil {
			log.Fatalf("Error parsing db file: %s\n", err)
		}

		tradb.Update()
		tradb.Write()

		bs, err := ioutil.ReadFile(db.TRADB)
		if err != nil {
			log.Fatalf("error reading file: %s\n", err)
		}

		fmt.Println(string(bs))		// send db to stdout
	}
}

func usage() {
	log.Printf("Usage: trago server:dir client-dir\n\n")

	log.Printf("-s|--server <dir>\n    %s\n", serverUsage);
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
