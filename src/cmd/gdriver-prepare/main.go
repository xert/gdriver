package main

import (
	"fmt"
	"gdrive"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"util"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

const (
	workDir = "./work"
)

func usage() {
	program := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s account@gmail.com", program)
	os.Exit(-1)
}

func main() {
	var accountFrom string

	if len(os.Args) < 2 {
		usage()
	}

	accountFrom = os.Args[1]

	ctx := context.Background()

	b, err := ioutil.ReadFile(gdrive.FileClientSecret)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := gdrive.GetClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	workExists, err := util.FileExists(workDir)
	if err != nil {
		log.Fatal(err.Error())
	}

	if workExists {
		err = fmt.Errorf("Working directory %s already exists.\nUse %s migrate or delete %s", workDir, filepath.Base(os.Args[0]), workDir)
	} else {
		err = prepare(srv, accountFrom)
	}

	if err != nil {
		log.Fatal(err.Error())
	}

}
