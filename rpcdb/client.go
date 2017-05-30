// tra
package rpcdb

import (
	"bytes"
	"fmt"
	"io"
	"net/rpc"
	"net/http"
	"sync"

	"github.com/ssbl/trago/db"
)


func Run(localDir, localAddr, remoteDir, remoteAddr string) error {
	var s    string
	var args int				// unused arg variables

	localClient, err := rpc.DialHTTP("tcp", localAddr)
	if err != nil {
		return err
	}
	remoteClient, err := rpc.DialHTTP("tcp", remoteAddr)
	if err != nil {
		return err
	}
	defer remoteClient.Call("TraSrv.StopSrv", &s, &args)

	err = startSrv(localClient, localDir)
	if err != nil {
		return err
	}
	err = startSrv(remoteClient, remoteDir)
	if err != nil {
		return err
	}

	localDb := db.TraDb{}
	err = localClient.Call("TraSrv.GetDb", &args, &localDb)
	if err != nil {
		return err
	}

	fmt.Printf("Reply from local trasrv:\n%v\n", localDb)

	remoteDb := db.TraDb{}
	err = remoteClient.Call("TraSrv.GetDb", &args, &remoteDb)
	if err != nil {
		return err
	}

	fmt.Printf("Reply from remote trasrv:\n%v\n", remoteDb)

	tags1, err := localDb.Compare(&remoteDb)
	if err != nil {
		return err
	}
	tags2, err := remoteDb.Compare(&localDb)
	if err != nil {
		return err
	}

	errch := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		processFileTags(localClient, localDb, remoteAddr, tags1, errch)
	}()

	go func() {
		defer wg.Done()
		processFileTags(remoteClient, remoteDb, localAddr, tags2, errch)
	}()
	wg.Wait()

	select {
		case err := <-errch:
		    return err
		default:
	}

	db.CombineVectors(localDb.VersionVec, remoteDb.VersionVec)
	remoteDb.VersionVec = localDb.VersionVec
	fmt.Println(localDb.VersionVec, remoteDb.VersionVec)

	if err := localClient.Call("TraSrv.PutDb", &localDb, &args); err != nil {
		return err
	}
	if err := remoteClient.Call("TraSrv.PutDb", &remoteDb, &args); err != nil {
		return err
	}

	return nil
}

func processFileTags(
	client *rpc.Client,
	tradb db.TraDb,
	dest string,
	tags map[string]db.FileTag,
	errch chan error,
) {
	var args int

	for file, tag := range tags {
		if tag == db.File {
			if err := sendFile(client, file, dest); err != nil {
				errch <- err
				return
			}
		} else if tag == db.Deleted {
			err := client.Call("TraSrv.RemoveFile", &file, &args)
			if err != nil {
				errch <- err
				return
			}
			delete(tradb.Files, file)
		} else if tag == db.Conflict {
			err := client.Call("TraSrv.ShowConflict", &file, &args)
			if err != nil {
				errch <- err
				return
			}
		}
	}
}

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	return client.Call("TraSrv.InitDb", &dir, &reply)
}

func sendFile(client *rpc.Client, file string, addr string) error {
	var reply int

	response, err := http.Get("http://"+addr+"/files/"+file)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return err
	}

	fileData := db.FileData{Name: file, Data: buf.Bytes()}
	return client.Call("TraSrv.PutFile", &fileData, &reply)
}
