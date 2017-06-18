// tra
package rpcdb

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		time.Sleep(time.Second)
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

	remoteDb := db.TraDb{}
	err = remoteClient.Call("TraSrv.GetDb", &args, &remoteDb)
	if err != nil {
		return err
	}

	fmt.Println("Local vs remote:")
	tags1, err := localDb.Compare(&remoteDb)
	if err != nil {
		return err
	}
	fmt.Println("Remote vs local:")
	tags2, err := remoteDb.Compare(&localDb)
	if err != nil {
		return err
	}

	err = checkTagsAndMerge(localClient, localDb, remoteDb, remoteAddr, tags1)
	if err != nil {
		return err
	}
	err = checkTagsAndMerge(remoteClient, remoteDb, localDb, localAddr, tags2)
	if err != nil {
		return err
	}

	if err := localClient.Call("TraSrv.PutDb", &localDb, &args); err != nil {
		return err
	}
	if err := remoteClient.Call("TraSrv.PutDb", &remoteDb, &args); err != nil {
		return err
	}

	return nil
}

func checkTagsAndMerge(
	client *rpc.Client,
	tradb db.TraDb,
	otherDb db.TraDb,
	dest string,
	tags db.TagList,
) error {
	var args int
	var err error
	var dirs []string

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
				mode := os.FileMode(tradb.Files[dir].Mode)
				if mode > 0 && !mode.IsDir() {
					// File was changed to directory.
					// Try to remove it and create the new directory.
					err = client.Call("TraSrv.RemoveFile", &dir, &args)
					if err != nil {
						return err
					}
				}
				dirData := db.FileData{Name: dir, Data: nil, Mode: tag.Mode}
				err = client.Call("TraSrv.PutDir", &dirData, &args)
				if err != nil {
					return err
				}
				delete(tags.Dirs, dir)
				tradb.Files[dir] = otherDb.Files[dir]
			}
		}
	}

	for file, tag := range tags.Files {
		switch tag.Label {
		case db.File:
			if os.FileMode(tradb.Files[file].Mode).IsDir() {
				// TODO: Find a better way to do this.
				// Directory was changed to a file.
				dirs = append(dirs, file)
				continue
			}

			if err := getFile(client, file, tag.Mode, dest); err != nil {
				return err
			}
			tradb.Files[file] = otherDb.Files[file]
		case db.Deleted:
			err = client.Call("TraSrv.RemoveFile", &file, &args)
			if err != nil {
				return err
			}
			delete(tradb.Files, file)
		case db.Conflict:
			err = client.Call("TraSrv.ShowConflict", &file, &args)
			if err != nil {
				return err
			}
		}
	}

	for _, dir := range dirs {
		tag := tags.Files[dir]

		dirData := db.FileData{Name: dir, Data: nil, Mode: 0}
		err = client.Call("TraSrv.RemoveDir", &dirData, &args)
		if err != nil {
			return err
		}

		err = getFile(client, dir, tag.Mode, dest)
		if err != nil {
			return err
		}
		tradb.Files[dir] = otherDb.Files[dir]
	}

	// Check if any directories have been deleted.
	// These directories should be empty at this stage.
	for level := maxDepth; level >= 0; level-- {
		for dir, tag := range tags.Dirs {
			dirLevel := strings.Count(dir, string(filepath.Separator))
			if tag.Label == db.Deleted && dirLevel == level {
				dirData := db.FileData{Name: dir, Data: nil, Mode: 0}
				err = client.Call("TraSrv.RemoveDir", &dirData, &args)
				if err != nil {
					return err
				}
				delete(tradb.Files, dir)
			}
		}
	}

	db.MergeVectors(tradb.VersionVec, otherDb.VersionVec)

	return err
}

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	return client.Call("TraSrv.InitDb", &dir, &reply)
}

func getFile(
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
