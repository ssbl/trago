package main

import (
	"log"

	"github.com/ssbl/trago/rpcdb"
)

const (
	// Test directories, addresses
	LOCALDIR  = "../a"
	REMOTEDIR = "../b"
	LOCALSRV  = "localhost:8999"
	REMOTESRV = "localhost:8998"
)


func main() {
	// TODO: Start trasrvs
	// For now, we manually start two servers (server.go) on the ports above.

	if err := rpcdb.Run(LOCALDIR, LOCALSRV, REMOTEDIR, REMOTESRV); err != nil {
		log.Fatal(err)
	}
}
