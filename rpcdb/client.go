// tra
package rpcdb

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/rpc"
	"path/filepath"
	"strings"

	"github.com/ssbl/trago/db"
)

func Run(localDir, localAddr, remoteDir, remoteAddr string) error {
	// Placeholder variables for RPC calls.
	var s string
	var args int

	localClient, err := rpc.DialHTTP("tcp", localAddr)
	if err != nil {
		return err
	}

	fmt.Println("Connecting to remote...")
retry:
	remoteClient, err := rpc.DialHTTP("tcp", remoteAddr)
	if err != nil {
		goto retry
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

	err = checkTags(localClient, localDb, remoteAddr, tags1)
	if err != nil {
		return err
	}
	err = checkTags(remoteClient, remoteDb, localAddr, tags2)
	if err != nil {
		return err
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

func checkTags(
	client *rpc.Client,
	tradb db.TraDb,
	dest string,
	tags db.TagList,
) error {
	var args int

	// Check if there are new directories.
	// First get the directory depth.
	maxDepth := 0
	for dir, _ := range tags.Dirs {
		depth := strings.Count(dir, string(filepath.Separator))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	// TODO: Use a better algorithm!
	for level := 0; level <= maxDepth; level++ {
		for dir, tag := range tags.Dirs {
			dirLevel := strings.Count(dir, string(filepath.Separator))
			if tag.Label == db.Directory && dirLevel == level {
				dirData := db.FileData{Name: dir, Data: nil, Mode: tag.Mode}
				err := client.Call("TraSrv.PutDir", &dirData, &args)
				if err != nil {
					return err
				}
				delete(tags.Dirs, dir)
			}
		}
	}

	for file, tag := range tags.Files {
		label := tag.Label

		if label == db.File {
			if err := sendFile(client, file, tag.Mode, dest); err != nil {
				return err
			}
		} else if label == db.Deleted {
			err := client.Call("TraSrv.RemoveFile", &file, &args)
			if err != nil {
				return err
			}
			delete(tradb.Files, file)
		} else if label == db.Conflict {
			err := client.Call("TraSrv.ShowConflict", &file, &args)
			if err != nil {
				return err
			}
		}
	}

	// Check if any directories have been deleted.
	// These directories should be empty at this stage.
	for level := maxDepth; level >= 0; level-- {
		for dir, tag := range tags.Dirs {
			fmt.Println("deleting directory", dir)
			dirLevel := strings.Count(dir, string(filepath.Separator))
			if tag.Label == db.Deleted && dirLevel == level {
				dirData := db.FileData{Name: dir, Data: nil, Mode: 0}
				err := client.Call("TraSrv.RemoveDir", &dirData, &args)
				if err != nil {
					return err
				}
				delete(tradb.Files, dir)
			}
		}
	}

	return nil
}

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	return client.Call("TraSrv.InitDb", &dir, &reply)
}

func sendFile(
	client *rpc.Client, file string, mode uint32, addr string,
) error {
	var args int

	response, err := http.Get("http://" + addr + "/files/" + file)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return err
	}

	fileData := db.FileData{Name: file, Data: buf.Bytes(), Mode: mode}
	return client.Call("TraSrv.PutFile", &fileData, &args)
}
