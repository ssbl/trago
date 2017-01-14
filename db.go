package main

import (
	"fmt"
	"io/ioutil"
	"os"
	// "path/filepath"
)

func main() {
	findFiles("./")
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
