// tra
package main

import (
	"fmt"
	"log"
	"net/rpc"

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
		fmt.Println("failed here")
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
	fmt.Println(localDb.Compare(&remoteDb))
	fmt.Println("Comparing remote with local...")
	fmt.Println(remoteDb.Compare(&localDb))
}

func startSrv(client *rpc.Client, dir string) error {
	var reply int

	err := client.Call("TraSrv.InitSrv", &dir, &reply)
	if err == db.FileNotFound {
		return nil
	}
	return err
}
