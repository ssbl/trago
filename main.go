package main

import (
	"flag"
	"log"
	"os"
	"strings"

	// "github.com/ssbl/trago/db"
)

const (
	SERVFLAG = "-s"
	SERVCMD = "trago -s {dir}"
	serverUsage = "Run in server mode in the specified directory.\n" +
		"If this is set, all other options are ignored."
	serverUsageShort = "Shorthand for --server."
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
	flag.StringVar(&flagDir, "s", defaultDir, serverUsageShort)

	log.SetFlags(0)
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if flagDir == defaultDir {
		server, serverDir, clientDir := parseArgs()
		log.Printf("%s:%s %s\n", server, serverDir, clientDir)
	}
}

func usage() {
	log.Printf("Usage: trago server:dir client-dir\n\n")

	flag.VisitAll(func(f *flag.Flag){
		log.Printf("-%s: %s\n", f.Name, f.Usage)
	})
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
