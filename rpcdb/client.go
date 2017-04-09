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
	args := 1 					// unused arg variable

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
		reply := 1
		if tag == db.File {
			response, err := http.Get("http://"+remoteAddr+"/files/"+file)
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
			err = localClient.Call("TraSrv.PutFile", &fileData, &reply)
			if err != nil {
				return err
			}

			localDb.Files[file] = remoteDb.Files[file]
		} else if tag == db.Deleted {
			err = localClient.Call("TraSrv.RemoveFile", &file, &reply)
			if err != nil {
				return err
			}
			delete(localDb.Files, file)
		} else if tag == db.Conflict {
			// TODO: send the file with a different name
			err = localClient.Call("TraSrv.ShowConflict", &file, &reply)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Comparing remote with local...")
	tags = remoteDb.Compare(&localDb)

	for file, tag := range tags {
		reply := 1
		if tag == db.File {
			response, err := http.Get("http://"+localAddr+"/files/"+file)
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
			err = remoteClient.Call("TraSrv.PutFile", &fileData, &reply)
			if err != nil {
				return err
			}
		} else if tag == db.Deleted {
			err = remoteClient.Call("TraSrv.RemoveFile", &file, &reply)
			if err != nil {
				return err
			}
			delete(remoteDb.Files, file)
		} else if tag == db.Conflict {
			err = remoteClient.Call("TraSrv.ShowConflict", &file, &reply)
			if err != nil {
				return err
			}
		} else if tag == db.Conflict {
			err = remoteClient.Call("TraSrv.ShowConflict", &file, &reply)
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
