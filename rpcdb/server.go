// trasrv
package main

import (
	"errors"
	"log"
	"net/rpc"
	"net/http"
	"os"

	"github.com/ssbl/trago/db"
)

type TraSrv int

var (
	localDb     *db.TraDb
	initialized bool
)

const (
	PORT = ":8999"
)


func (t *TraSrv) GetDb(args *int, reply *db.TraDb) error {
	if !initialized {
		return errors.New("uninitialized")
	} else {
		*reply = *localDb
		return nil
	}
}

func (t *TraSrv) PutDb(args *db.TraDb, reply *int) error {
	localDb = &db.TraDb{}
	*localDb = *args
	return nil
}

func (t *TraSrv) PutFile(data *db.FileData, reply *int) error {
	file, err := os.Create(data.Name)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data.Data)
	return err
}

func (t *TraSrv) InitSrv(dir *string, reply *int) error {
	err := os.Chdir(*dir)
	if err != nil {
		return err
	}

	localDb, err = db.ParseFile()
	if err == db.FileNotFound {
		initialized = true
		return nil
	} else if err != nil {
		return err
	}

	initialized = true
	return nil
}

func main() {
	trasrv := new(TraSrv)
	rpc.Register(trasrv)
	rpc.HandleHTTP()

	port := ":" + os.Args[1]

	http.Handle("/files/", http.StripPrefix("/files/",
		http.FileServer(http.Dir("."))))
	log.Fatal(http.ListenAndServe(port, nil))
}

func assert(err error, format string, args ...interface{}) {
	if err != nil {
		log.Fatalf(format, args)
	}
}
