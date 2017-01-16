package main

import (
	"math/rand"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const (
	TRADB = ".trago.db"
	bytes = "abcdefghijklmnopqrstuvwxyz1234567890"
)

type TraDb struct {
	replicaId string
	version int
	files map[string]FileState
}

type FileState struct {
	size int
	mtime time.Time
	// TODO: use a hash as well
}

func main() {
	db, _ := parseDbFile()

	fmt.Println(db)
}

func parseDbFile() (TraDb, error) {
	tradb := TraDb{}

	_, err := ioutil.ReadFile(TRADB)
	if os.IsNotExist(err) {
		fmt.Println("didn't find .trago.db")
		tradb = createDb()
	} else {
		fmt.Println(err)
	}

	return tradb, nil
}

func createDb() TraDb {
	replicaId := make([]byte, 16)
	version := 

	replicaId := make([]byte, 16)
	version := 1

	for i, _ := range replicaId {
		replicaId[i] = bytes[rand.Intn(len(bytes))]
	}

	return TraDb{}
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
