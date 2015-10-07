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
	fmt.Printf("Usage: %s ID1 ID2", program)
	os.Exit(-1)
}

func main() {
	var id1, id2 string

	if len(os.Args) < 2 {
		usage()
	}

	if len(os.Args) != 3 {
		usage()
	}
	id1 = os.Args[1]
	id2 = os.Args[2]

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

	err = compare(srv, id1, id2)

	if err != nil {
		log.Fatal(err.Error())
	}

}
