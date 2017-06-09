package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func ReadDir(dir string) (map[string]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)

	if err := readDir(dir, files); err != nil {
		return nil, err
	}

	return files, nil
}

func readDir(dir string, filemap map[string]os.FileInfo) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, fileinfo := range files {
		name := filepath.Join(dir, fileinfo.Name())

		filemap[name] = fileinfo
		if fileinfo.IsDir() {
			err := readDir(name, filemap)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
