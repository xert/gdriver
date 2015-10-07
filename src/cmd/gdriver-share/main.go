package main

import (
	"fmt"
	"gdrive"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

func usage() {
	program := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s owner@gmail.com shareto-account@gmail.com\n", program)
	os.Exit(-1)
}

func main() {
	var accountFrom string
	var accountTo string

	if len(os.Args) < 3 {
		usage()
	}

	accountFrom = os.Args[1]
	accountTo = os.Args[2]

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

	log.Printf("Sharing files owned by %s to %s", accountFrom, accountTo)

	err = share(srv, accountFrom, accountTo)

	if err != nil {
		log.Fatal(err.Error())
	}

}
