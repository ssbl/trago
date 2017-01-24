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
	ReplicaId string
	Version   map[string]int
	Files     map[string]FileState
}

type FileState struct {
	Size    int
	MTime   int64
	Version int
	Replica string
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
	tradb.Files = make(map[string]FileState)

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

			tradb.Files[fields[1]] = FileState{size, mtime, ver, replicaId}
		case "version": // version r1:v1 r2:v2 ...
			for _, entry := range fields[1:] {
				pair := strings.Split(entry, ":") // replica:version pair

				v, err := strconv.Atoi(pair[1])
				checkError(err)

				version[pair[0]] = v
			}
			tradb.Version = version

		case "replica": // replica replica-id
			if len(fields) != 2 {
				continue
			}
			tradb.ReplicaId = fields[1]
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
			Size:    int(file.Size()),
			MTime:   file.ModTime().UnixNano(),
			Version: 1,
			Replica: string(replicaId),
		}
		filemap[file.Name()] = fs
	}

	return &TraDb{string(replicaId), version, filemap}
}

func (tradb *TraDb) Write() {
	var pairs []string

	for replicaId, version := range tradb.Version {
		entry := strings.Join([]string{replicaId, strconv.Itoa(version)}, ":")
		pairs = append(pairs, entry)
	}

	versionVector := strings.Join(pairs, " ")

	preamble := fmt.Sprintf(
		"replica %s\nversion %s\n# files\n",
		tradb.ReplicaId,
		versionVector,
	)

	fileEntries := make([]string, len(tradb.Files))

	i := 0
	for filename, info := range tradb.Files {
		fileEntries[i] = fmt.Sprintf(
			"file %s %d %d %s:%d",
			filename,
			info.Size,
			info.MTime,
			info.Replica,
			info.Version,
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
		dbRecord := db.Files[filename]
		if dbRecord.MTime == 0 {
			log.Printf("found a new file: %s\n", filename)
		} else if dbRecord.MTime < file.ModTime().UnixNano() {
			log.Printf("found an updated file: %s\n", filename)
			dbRecord.MTime = file.ModTime().UnixNano()
			dbRecord.Version = db.Version[db.ReplicaId]
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
