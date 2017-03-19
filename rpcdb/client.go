// tra
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"net/http"
	"os"

	"github.com/ssbl/trago/db"
)


const (
	LOCALDIR  = "../a"
	REMOTEDIR = "../b"
)	

func main() {
	localClient, err := rpc.DialHTTP("tcp", "localhost:8999")
	if err != nil {
		log.Fatal(err)
	}
	remoteClient, err := rpc.DialHTTP("tcp", "localhost:8998")
	if err != nil {
		log.Fatal(err)
	}

	err = startSrv(localClient, LOCALDIR)
	if err != nil {
		log.Fatal(err)
	}
	err = startSrv(remoteClient, REMOTEDIR)
	if err != nil {
		log.Fatal(err)
	}

	args := 1
	localDb := db.TraDb{}
	err = localClient.Call("TraSrv.GetDb", &args, &localDb)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reply from local trasrv:\n%v\n", localDb)

	remoteDb := db.TraDb{}
	err = remoteClient.Call("TraSrv.GetDb", &args, &remoteDb)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reply from remote trasrv:\n%v\n", remoteDb)

	fmt.Println("Comparing local with remote...")
	tags := localDb.Compare(&remoteDb)

	for file, tag := range tags {
		if tag == db.File {
			response, err := http.Get("http://localhost:8998/files/"+file)
			if err != nil {
				log.Fatal(err)
			}
			defer response.Body.Close()

			fmt.Printf("Got file %s:\n", file)
			io.Copy(os.Stdout, response.Body)

			reply := 1
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, response.Body)
			if err != nil {
				log.Fatal(err)
			}

			fileData := db.FileData{Name: file, Data: buf.Bytes()}
			err = localClient.Call("TraSrv.PutFile", &fileData, &reply)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	err := client.Call("TraSrv.InitSrv", &dir, &reply)
	if err == db.FileNotFound {
		return nil
	}
	return err
}
