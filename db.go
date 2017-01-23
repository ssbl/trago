package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

const (
	TRADB      = ".trago.db"
	bytes      = "abcdefghijklmnopqrstuvwxyz1234567890"
	currentDir = "./"
)

type TraDb struct {
	replicaId string
	version   map[string]int
	files     map[string]FileState
}

type FileState struct {
	size    int
	mtime   int64
	version int
	replica string
	// TODO: use a hash as well
}

func main() {
	db, err := Parse()
	checkError(err)

	db.Update()
}

func Parse() (TraDb, error) {
	tradb := TraDb{}
	version := make(map[string]int)

	dbfile, err := os.Open(TRADB)
	if os.IsNotExist(err) {
		log.Println("didn't find .trago.db")
		tradb = *New()
		tradb.Write()

		return tradb, nil
	} else if err != nil {
		return tradb, err
	}

	defer dbfile.Close()
	tradb.files = make(map[string]FileState)

	scanner := bufio.NewScanner(dbfile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "file": // file name size mtime replica:version
			if len(fields) != 5 {
				continue
			}

			size, err := strconv.Atoi(fields[2])
			checkError(err)
			mtime, err := strconv.ParseInt(fields[3], 10, 64)
			checkError(err)

			pair := strings.Split(fields[4], ":")
			replicaId := pair[0]
			ver, err := strconv.Atoi(pair[1])
			checkError(err)

			tradb.files[fields[1]] = FileState{size, mtime, ver, replicaId}
		case "version": // version r1:v1 r2:v2 ...
			for _, entry := range fields[1:] {
				pair := strings.Split(entry, ":") // replica:version pair

				v, err := strconv.Atoi(pair[1])
				checkError(err)

				version[pair[0]] = v
			}
			tradb.version = version

		case "replica": // replica replica-id
			if len(fields) != 2 {
				continue
			}
			tradb.replicaId = fields[1]
		}
	}

	checkError(scanner.Err())

	return tradb, nil
}

func New() *TraDb {
	replicaId := make([]byte, 16)
	version := make(map[string]int)

	for i, _ := range replicaId {
		replicaId[i] = bytes[rand.Intn(len(bytes))]
	}
	version[string(replicaId)] = 1

	files, err := ioutil.ReadDir(currentDir)
	checkError(err)

	filemap := make(map[string]FileState)
	for _, file := range files {
		if file.IsDir() {
			continue // ignore directories for now
		}
		fs := FileState{
			size:    int(file.Size()),
			mtime:   file.ModTime().UnixNano(),
			version: 1,
			replica: string(replicaId),
		}
		filemap[file.Name()] = fs
	}

	return &TraDb{string(replicaId), version, filemap}
}

func (tradb *TraDb) Write() {
	var pairs []string

	for replicaId, version := range tradb.version {
		entry := strings.Join([]string{replicaId, strconv.Itoa(version)}, ":")
		pairs = append(pairs, entry)
	}

	versionVector := strings.Join(pairs, " ")

	preamble := fmt.Sprintf(
		"replica %s\nversion %s\n# files\n",
		tradb.replicaId,
		versionVector,
	)

	fileEntries := make([]string, len(tradb.files))

	i := 0
	for filename, info := range tradb.files {
		fileEntries[i] = fmt.Sprintf(
			"file %s %d %d %s:%d",
			filename,
			info.size,
			info.mtime,
			info.replica,
			info.version,
		)
		i = i + 1
	}

	entryString := strings.Join(fileEntries, "\n")
	dataToWrite := []byte(preamble + entryString)

	err := ioutil.WriteFile(TRADB, dataToWrite, 0644)
	checkError(err)
}

func (db *TraDb) Update() {
	files, err := ioutil.ReadDir(currentDir)
	checkError(err)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		dbRecord := db.files[filename]
		if dbRecord.mtime == 0 {
			log.Printf("found a new file: %s\n", filename)
		} else if dbRecord.mtime < file.ModTime().UnixNano() {
			log.Printf("found an updated file: %s\n", filename)
			dbRecord.mtime = file.ModTime().UnixNano()
			dbRecord.version = db.version[db.replicaId]
		} else {
			log.Printf("file unchanged: %s\n", file.Name())
		}
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
