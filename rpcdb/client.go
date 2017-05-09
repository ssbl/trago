// tra
package rpcdb

import (
	"bytes"
	"fmt"
	"io"
	"net/rpc"
	"net/http"

	"github.com/ssbl/trago/db"
)


func Run(localDir, localAddr, remoteDir, remoteAddr string) error {
	var args int 					// unused arg variable

	localClient, err := rpc.DialHTTP("tcp", localAddr)
	if err != nil {
		return err
	}
	remoteClient, err := rpc.DialHTTP("tcp", remoteAddr)
	if err != nil {
		return err
	}

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

	fmt.Println("Comparing local with remote...")
	tags := localDb.Compare(&remoteDb)

	for file, tag := range tags {
		if tag == db.File {
			if err := sendFile(localClient, file, remoteAddr); err != nil {
				return err
			}
			localDb.Files[file] = remoteDb.Files[file]
		} else if tag == db.Deleted {
			err = localClient.Call("TraSrv.RemoveFile", &file, &args)
			if err != nil {
				return err
			}
			delete(localDb.Files, file)
		} else if tag == db.Conflict {
			err = localClient.Call("TraSrv.ShowConflict", &file, &args)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Comparing remote with local...")
	tags = remoteDb.Compare(&localDb)

	for file, tag := range tags {
		if tag == db.File {
			if err := sendFile(remoteClient, file, localAddr); err != nil {
				return err
			}
		} else if tag == db.Deleted {
			err = remoteClient.Call("TraSrv.RemoveFile", &file, &args)
			if err != nil {
				return err
			}
			delete(remoteDb.Files, file)
		} else if tag == db.Conflict {
			err = remoteClient.Call("TraSrv.ShowConflict", &file, &args)
			if err != nil {
				return err
			}
		}
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

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	err := client.Call("TraSrv.InitSrv", &dir, &reply)
	return err
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
	err = client.Call("TraSrv.PutFile", &fileData, &reply)

	return err
}
