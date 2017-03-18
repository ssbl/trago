// tra
package main

import (
	"net/rpc"
	"log"

	"github.com/ssbl/trago/db"
)


func main() {

	client, err := rpc.DialHTTP("tcp", "localhost:8999")
	if err != nil {
		log.Fatal(err)
	}

	args := 1
	reply := db.TraDb{}
	err = client.Call("TraSrv.GetDb", &args, reply)
	if err != nil {
		log.Fatal(err)
	}
}
