// trasrv
package rpcdb

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

	if err := localDb.UpdateMTimes(); err != nil {
		return err
	}

	return localDb.Write()
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

func (t *TraSrv) RemoveFile(filename *string, reply *int) error {
	return os.Remove(*filename)
}

func (t *TraSrv) ShowConflict(filename *string, reply *int) error {
	log.Printf("Conflict: %s\n", *filename)
	return nil
}

func (t *TraSrv) InitDb(dir *string, reply *int) error {
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

	localDb.VersionVec[localDb.ReplicaId] += 1

	log.Println(localDb)
	if err := localDb.Update(); err != nil {
		return err
	}

	initialized = true
	return nil
}

func Start(port string) error {
	trasrv := new(TraSrv)

	if err := rpc.Register(trasrv); err != nil {
		return err
	}

	rpc.HandleHTTP()

	http.Handle("/files/", http.StripPrefix("/files/",
		http.FileServer(http.Dir("."))))

	return http.ListenAndServe(port, nil)
}
