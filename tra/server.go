// trasrv
package tra

import (
	"errors"
	"log"
	"net/http"
	"net/rpc"
	"os"

	"github.com/ssbl/trago/db"
)

type TraSrv struct {
	localDb     *db.TraDb
	initialized bool
}

func (t *TraSrv) GetDb(args *int, reply *db.TraDb) error {
	if !t.initialized {
		return errors.New("uninitialized")
	} else {
		*reply = *t.localDb
		return nil
	}
}

func (t *TraSrv) PutDb(args *db.TraDb, reply *int) error {
	t.localDb = &db.TraDb{}
	*t.localDb = *args

	if err := t.localDb.UpdateMTimes(); err != nil {
		return err
	}

	return t.localDb.WriteToFile()
}

func (t *TraSrv) PutFile(data *db.FileData, args *int) error {
	perm := os.FileMode(data.Mode) & os.ModePerm
	file, err := os.OpenFile(data.Name, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer func() { err = file.Close() }()

	if err := file.Chmod(perm); err != nil {
		return err
	}

	_, err = file.Write(data.Data)
	return err
}

func (t *TraSrv) PutDir(dir *db.FileData, args *int) error {
	err := os.Mkdir(dir.Name, os.FileMode(dir.Mode)&os.ModePerm)
	if os.IsExist(err) {
		return nil
	}
	return err
}

func (t *TraSrv) RemoveDir(dir *db.FileData, args *int) error {
	return os.Remove(dir.Name)
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

	t.localDb, err = db.ParseFile()
	if err != nil {
		return err
	}

	t.localDb.VersionVec[t.localDb.ReplicaId]++

	if err := t.localDb.Update(); err != nil {
		return err
	}

	t.initialized = true
	return nil
}

func (t *TraSrv) StopSrv(arg *string, reply *int) error {
	defer os.Exit(0)
	return nil
}

func StartSrv(port string) error {
	trasrv := new(TraSrv)

	if err := rpc.Register(trasrv); err != nil {
		return err
	}

	rpc.HandleHTTP()

	http.Handle("/files/", http.StripPrefix("/files/",
		http.FileServer(http.Dir("."))))

	return http.ListenAndServe(port, nil)
}
