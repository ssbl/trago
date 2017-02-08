package db

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	TRADB      = ".trago.db"
	bytes      = "abcdefghijklmnopqrstuvwxyz1234567890"
	currentDir = "./"
)

type TraDb struct {
	ReplicaId  string
	VersionVec map[string]int
	Files      map[string]FileState
}

type FileState struct {
	Size    int
	MTime   int64
	Version int
	Replica string
	// TODO: use a hash as well
}

func Parse(data string) (TraDb, error) {
	tradb := TraDb{}
	versionVector := make(map[string]int)

	tradb.Files = make(map[string]FileState)

	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)

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
			if err != nil {
				return tradb, err
			}

			mtime, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return tradb, err
			}

			pair := strings.Split(fields[4], ":")
			replicaId := pair[0]
			ver, err := strconv.Atoi(pair[1])
			if err != nil {
				return tradb, err
			}

			tradb.Files[fields[1]] = FileState{size, mtime, ver, replicaId}
		case "version": // version r1:v1 r2:v2 ...
			for _, entry := range fields[1:] {
				pair := strings.Split(entry, ":") // replica:version pair

				v, err := strconv.Atoi(pair[1])
				if err != nil {
					return tradb, err
				}

				versionVector[pair[0]] = v
			}
			tradb.VersionVec = versionVector

		case "replica": // replica replica-id
			if len(fields) != 2 {
				continue
			}
			tradb.ReplicaId = fields[1]
		}
	}

	return tradb, nil
}

func ParseFile() (TraDb, error) {
	tradb := TraDb{}

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

	bs, err := ioutil.ReadFile(TRADB)
	if err != nil {
		return tradb, err
	}

	return Parse(string(bs))
}

func New() *TraDb {
	replicaId := make([]byte, 16)
	versionVector := make(map[string]int)

	rand.Seed(time.Now().UTC().UnixNano())
	for i, _ := range replicaId {
		replicaId[i] = bytes[rand.Intn(len(bytes))]
	}
	versionVector[string(replicaId)] = 1

	files, err := ioutil.ReadDir(currentDir)
	checkError(err)

	filemap := make(map[string]FileState)
	for _, file := range files {
		if file.IsDir() {
			continue // ignore directories for now
		}
		fs := FileState{
			Size:    int(file.Size()),
			MTime:   file.ModTime().UTC().UnixNano(),
			Version: 1,
			Replica: string(replicaId),
		}
		filemap[file.Name()] = fs
	}

	return &TraDb{string(replicaId), versionVector, filemap}
}

func (tradb *TraDb) Write() error {
	var pairs []string

	for replicaId, version := range tradb.VersionVec {
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
	return err
}

func (db *TraDb) Update() error {
	files, err := ioutil.ReadDir(currentDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		dbRecord := db.Files[filename]
		if dbRecord.MTime == 0 {
			log.Printf("found a new file: %s\n", filename)
		} else if dbRecord.MTime < file.ModTime().UTC().UnixNano() {
			log.Printf("found an updated file: %s\n", filename)
			dbRecord.MTime = file.ModTime().UTC().UnixNano()
			dbRecord.Version = db.VersionVec[db.ReplicaId]
		} else {
			log.Printf("file unchanged: %s\n", filename)
		}
	}

	db.VersionVec[db.ReplicaId] += 1
	return nil
}

func (local *TraDb) Compare(remote *TraDb) {
	remoteFiles := remote.Files

	for file, state := range local.Files {
		remoteState := remoteFiles[file]

		if remoteState.Version == 0 { // file not present on server
			// TODO: download only if we have a more "recent" copy
			continue
		}

		if isFileChanged(state, remoteState) {
			if local.VersionVec[remoteState.Replica] >= remoteState.Version {
				continue // we already know about changes on remote
			} else if remote.VersionVec[state.Replica] >= state.Version {
				log.Printf("downloading: %s\n", file)
				continue
			} else {
				log.Printf("conflict: %s\n", file)
			}
		}
	}
}

func isFileChanged(fs1 FileState, fs2 FileState) bool {
	if fs1.MTime != fs2.MTime || fs1.Size != fs2.Size {
		return false
	}
	return true
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
