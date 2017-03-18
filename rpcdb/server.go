// trasrv
package main

import (
	"net/rpc"
	"net/http"
	"errors"
	"log"

	"github.com/ssbl/trago/db"
)


type TraSrv int

func (t *TraSrv) GetDb(args *int, reply *db.TraDb) error {
	return errors.New("error fetching db")
}

func main() {
	trasrv := new(TraSrv)
	rpc.Register(trasrv)
	rpc.HandleHTTP()

	log.Fatal(http.ListenAndServe(":8999", nil))
}
