package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
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

	if !isServer() {
		server, serverDir, clientDir := parseArgs()
		fmt.Printf("%s:%s %s\n", server, serverDir, clientDir)

		err := os.Chdir(clientDir)
		assert(err, "Error changing to directory: %s\n", err)

		cmd := exec.Command("ssh", server,
			strings.Replace(SERVCMD, "{dir}", serverDir, 1))

		stdin, err := cmd.StdinPipe()
		assert(err, "Error creating pipe: %s\n", err)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		localDb, err := db.ParseFile()
		assert(err, "Error parsing db file: %s\n", err)

		err = localDb.Update()
		assert(err, "Error updating local db: %s\n", err)

		err = cmd.Start()
		assert(err, stderr.String())

		fmt.Println("sending get")
		_, err = stdin.Write([]byte("get\n"))
		assert(err, "Error writing to pipe: %s\n", err)

		fmt.Println("waiting for reply")
		out := readStdout(stdout)

		remoteDb, err := db.Parse(out)
		assert(err, "Error parsing server response\n")

		err = localDb.Write()
		assert(err, "Error writing to db file: %s\n", err)

		fmt.Println(remoteDb)
		tags := localDb.Compare(remoteDb)

		_, err = stdin.Write([]byte("quit\n"))
		assert(err, "Error writing to pipe: %s\n", err)

		stdin.Close()
		fmt.Println(tags)
	} else {	  // running in server mode, so we ignore all other flags
		err := os.Chdir(flagDir)
		assert(err, "Error changing to directory: %s\n", err)

		tradb, err := db.ParseFile()
		assert(err, "Error parsing db file: %s\n", err)

		err = tradb.Update()
		assert(err, "Error updating db file: %s\n", err)

		err = tradb.Write()
		assert(err, "Error writing to db file: %s\n", err)

		bs, err := ioutil.ReadFile(db.TRADB)
		assert(err, "Error reading file: %s\n", err)

		cmdLoop(string(bs))
	}
}

func readStdout(stdout *bytes.Buffer) string {
	var buf [512]byte
	out := new(bytes.Buffer)

	for {
		n, _ := stdout.Read(buf[0:])

		out.Write(buf[0:n])
		outStr := out.String()

		if strings.HasSuffix(outStr, "\n\n\n") {
			return outStr
		}
	}
}

func cmdLoop(db string) {
	for {
		msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err == io.EOF {
			return
		}

		switch (strings.TrimSpace(msg)) {
		case "quit":
			fmt.Println("quitting")

		case "get":
			fmt.Println(db)
			fmt.Println("\n")
		}
	}
}

func isServer() bool {
	return flag.NFlag() == 1
}

func usage() {
	log.Printf("Usage: trago <server>:<dir> <client-dir>\n\n")

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
