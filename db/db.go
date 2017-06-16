package db

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ssbl/trago/fs"
)

const (
	TRADB      = ".trago.db"
	chars      = "abcdefghijklmnopqrstuvwxyz1234567890"
	currentDir = "."
)

var (
	ErrFileNotFound = errors.New("Couldn't find .trago.db")
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
	Hash    string
	Mode    uint32
}

type FileData struct {
	Name string
	Data []byte
	Mode uint32
}

type Label uint8

type FileTag struct {
	Label Label
	Mode  uint32
}

type TagList struct {
	Files map[string]FileTag
	Dirs  map[string]FileTag
}

const (
	File = Label(iota)
	Conflict
	Directory
	Deleted
)

// Parse parses a TraDb structure.
// Fails if the given string is not in the correct format.
func Parse(data string) (*TraDb, error) {
	tradb := &TraDb{}
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
		case "file": // file name size mtime replica:version hash mode
			if len(fields) != 7 {
				continue
			}

			size, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, err
			}

			mtime, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return nil, err
			}

			pair := strings.Split(fields[4], ":")
			replicaId := pair[0]
			ver, err := strconv.Atoi(pair[1])
			if err != nil {
				return nil, err
			}

			sum := fields[5]

			mode, err := strconv.ParseUint(fields[6], 10, 32)
			if err != nil {
				return nil, err
			}

			tradb.Files[fields[1]] = FileState{size, mtime, ver, replicaId, sum, uint32(mode)}
		case "version": // version r1:v1 r2:v2 ...
			for _, entry := range fields[1:] {
				pair := strings.Split(entry, ":") // replica:version pair

				v, err := strconv.Atoi(pair[1])
				if err != nil {
					return nil, err
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

// ParseFile parses a TraDb from the db file in the current directory.
func ParseFile() (*TraDb, error) {
	tradb := &TraDb{}

	dbfile, err := os.Open(TRADB)
	if os.IsNotExist(err) {
		log.Println(ErrFileNotFound.Error())
		tradb, err = New()
		if err == nil {
			return tradb, ErrFileNotFound
		} else {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	defer func() { err = dbfile.Close() }()

	bs, err := ioutil.ReadFile(TRADB)
	if err != nil {
		return nil, err
	}

	tradb, err = Parse(string(bs))
	return tradb, err
}

// New creates a new TraDb.
//
// The replica ID is a random string, and the version
// number is set to 1. Checks for files in the current
// directory and stores relevant file state in a map.
func New() (*TraDb, error) {
	replicaId := make([]byte, 16)
	versionVector := make(map[string]int)

	rand.Seed(time.Now().UTC().UnixNano())
	for i, _ := range replicaId {
		replicaId[i] = chars[rand.Intn(len(chars))]
	}
	versionVector[string(replicaId)] = 1 // TODO: check for duplicates

	files, err := fs.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}

	delete(files, TRADB)
	filemap := make(map[string]FileState)

	for filename, file := range files {
		var err error
		var hashString string

		if !file.IsDir() {
			hashString, err = hash(filename)
			if err != nil {
				return nil, err
			}
		} else {
			hashString = "[dir]"
		}

		fs := FileState{
			Size:    int(file.Size()),
			MTime:   file.ModTime().UTC().UnixNano(),
			Version: 1,
			Replica: string(replicaId),
			Hash:    hashString,
			Mode:    uint32(file.Mode()),
		}
		filemap[filename] = fs
	}

	return &TraDb{string(replicaId), versionVector, filemap}, nil
}

// WriteToFile writes a TraDb to the db file .trago.db.
func (tradb *TraDb) WriteToFile() error {
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
		var err error
		var hashString string

		if mode := os.FileMode(info.Mode); mode.IsDir() {
			hashString = "[dir]"
		} else {
			hashString, err = hash(filename)
			if err != nil {
				return err
			}
		}

		fileEntries[i] = fmt.Sprintf(
			"file %s %d %d %s:%d %s %d",
			filename,
			info.Size,
			info.MTime,
			info.Replica,
			info.Version,
			hashString,
			info.Mode,
		)
		i++
	}

	entryString := strings.Join(fileEntries, "\n")
	dataToWrite := []byte(preamble + entryString)

	return ioutil.WriteFile(TRADB, dataToWrite, 0644)
}

// Update looks for modified files in the current directory
// and updates the filemap accordingly.
func (db *TraDb) Update() error {
	files, err := fs.ReadDir(currentDir)
	if err != nil {
		return err
	}
	delete(files, TRADB)

	visitedFiles := make(map[string]bool)
	ourVersion := db.VersionVec[db.ReplicaId]

	for filename, file := range files {
		var err error
		var hashString string

		if file.IsDir() {
			hashString = "[dir]"
		} else {
			hashString, err = hash(filename)
			if err != nil {
				return err
			}
		}

		dbRecord := db.Files[filename]
		if dbRecord.Version == 0 {
			log.Printf("found a new file: %s\n", filename)

			db.Files[filename] = FileState{
				Size:    int(file.Size()),
				MTime:   file.ModTime().UTC().UnixNano(),
				Version: ourVersion,
				Replica: db.ReplicaId,
				Hash:    hashString,
				Mode:    uint32(file.Mode()),
			}
		} else if os.FileMode(dbRecord.Mode).IsDir() {
			visitedFiles[filename] = true
			continue
		} else if dbRecord.MTime < file.ModTime().UTC().UnixNano() ||
			os.FileMode(dbRecord.Mode) != file.Mode()&0777 {
			log.Printf("found an updated file: %s\n", filename)
			dbRecord.MTime = file.ModTime().UTC().UnixNano()
			dbRecord.Version = ourVersion
			dbRecord.Mode = uint32(file.Mode())
			dbRecord.Replica = db.ReplicaId
			dbRecord.Hash = hashString
			db.Files[filename] = dbRecord
		} else {
			log.Printf("file unchanged: %s\n", filename)
		}
		visitedFiles[filename] = true
	}

	// Check for deleted files and directories.
	for filename, _ := range db.Files {
		if !visitedFiles[filename] {
			log.Printf("update: deleting entry for %s\n", filename)
			delete(db.Files, filename)
		}
	}

	return nil
}

// Compare compares two TraDbs.
// Returns a TagList which gives the FileTag for each changed file.
func (local *TraDb) Compare(remote *TraDb) (TagList, error) {
	var tags TagList

	tags.Dirs = make(map[string]FileTag)
	tags.Files = make(map[string]FileTag)
	remoteFiles := remote.Files

	for file, state := range local.Files {
		isDir := os.FileMode(state.Mode).IsDir()
		remoteState := remoteFiles[file]

		// File or directory doesn't exist on the remote replica.
		if remoteState.Version == 0 {
			if state.Version <= remote.VersionVec[state.Replica] {
				log.Printf("deleting: %s\n", file)
				if isDir {
					tags.Dirs[file] = FileTag{Deleted, 0}
				} else {
					tags.Files[file] = FileTag{Deleted, 0}
				}
			}
			continue
		}

		if isDir {
			continue
		}

		changed, err := isFileChanged(state, remoteState)
		if err != nil {
			return tags, err
		}

		if changed {
			if local.VersionVec[remoteState.Replica] >= remoteState.Version {
				log.Printf("keeping: %s\n", file)
			} else if remote.VersionVec[state.Replica] >= state.Version {
				log.Printf("downloading: %s\n", file)
				tags.Files[file] = FileTag{File, remoteFiles[file].Mode}
			} else {
				log.Printf("conflict: %s\n", file)
				tags.Files[file] = FileTag{Conflict, 0}
			}
		} else {
			log.Printf("unchanged: %s\n", file)
		}
	}

	for file, state := range remoteFiles {
		if local.Files[file].Version > 0 {
			continue
		} else if state.Version > local.VersionVec[state.Replica] {
			if mode := os.FileMode(state.Mode); mode.IsDir() {
				log.Printf("new directory: %s\n", file)
				tags.Dirs[file] = FileTag{Directory, remoteFiles[file].Mode}
			} else {
				log.Printf("downloading new file: %s\n", file)
				tags.Files[file] = FileTag{File, remoteFiles[file].Mode}
			}
		}
	}

	return tags, nil
}

func MergeVectors(v1 map[string]int, v2 map[string]int) {
	for replica, version := range v1 {
		if v2[replica] > version {
			v1[replica] = v2[replica]
		}
	}

	for replica, version := range v2 {
		if v1[replica] < version {
			v1[replica] = version
		}
	}
}

func hash(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return filename, err
	}
	defer func() { err = f.Close() }()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return filename, err
	}

	return hex.EncodeToString(h.Sum(nil)), err
}

func isFileChanged(fs1 FileState, fs2 FileState) (bool, error) {
	if fs1.Mode != fs2.Mode {
		return true, nil
	}
	if fs1.MTime != fs2.MTime || fs1.Size != fs2.Size {
		h1, err := hex.DecodeString(fs1.Hash)
		if err != nil {
			return false, err
		}

		h2, err := hex.DecodeString(fs2.Hash)
		if err != nil {
			return false, err
		}

		return !bytes.Equal(h1, h2), nil
	}
	return false, nil
}
