package main

import (
	"math/rand"
	"fmt"
	"io/ioutil"
	"os"
	// "time"
)

const (
	TRADB = ".trago.db"
	bytes = "abcdefghijklmnopqrstuvwxyz1234567890"
	currentDir = "./"
)

type TraDb struct {
	replicaId string
	version int
	files map[string]FileState
}

type FileState struct {
	size int
	mtime int64
	// TODO: use a hash as well
}

func main() {
	db, err := parseDbFile()
	checkError(err)

	fmt.Println(db)
}

func parseDbFile() (TraDb, error) {
	tradb := TraDb{}

	_, err := ioutil.ReadFile(TRADB)
	if os.IsNotExist(err) {
		fmt.Println("didn't find .trago.db")
		tradb = createDb()
		writeDb(tradb)
	} else {
		return tradb, err
	}

	return tradb, nil
}

func createDb() TraDb {
	replicaId := make([]byte, 16)
	version := 1

	for i, _ := range replicaId {
		replicaId[i] = bytes[rand.Intn(len(bytes))]
	}

	files, err := ioutil.ReadDir(currentDir)
	checkError(err)

	filemap := make(map[string]FileState)
	for _, file := range files {
		if file.IsDir() {
			continue			// ignore directories for now
		}
		fs := FileState{
			size: int(file.Size()),
			mtime: file.ModTime().UnixNano(),
		}
		filemap[file.Name()] = fs
	}

	return TraDb{
		replicaId: string(replicaId),
		version: version,
		files: filemap,
	}
}

func findFiles(path string) {
	files, err := ioutil.ReadDir(path)
	checkError(err)

	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
