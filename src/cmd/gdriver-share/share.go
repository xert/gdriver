package main

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb"

	"google.golang.org/api/drive/v2"
)

func findAllFilesFrom(srv *drive.Service, owner string) ([]*drive.File, error) {
	var f []*drive.File
	var err error

	pageToken := ""
	for {
		q := srv.Files.List().Q("'" + owner + "' in owners").MaxResults(1000)
		// If we have a pageToken set, apply it to the query
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		var r *drive.FileList

		for i := 0; i < 10; i++ {
			r, err = q.Do()
			if err != nil {
				fmt.Printf("findAllFilesFrom: %s\n", err.Error())
				continue
			}
			break

		}
		if err != nil {
			return nil, err
		}

		f = append(f, r.Items...)

		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return f, nil
}

func shareFile(srv *drive.Service, file *drive.File, accountTo string) error {
	var err error

	p := &drive.Permission{Value: accountTo, Type: "user", Role: "reader"}

	for i := 0; i < 10; i++ {
		_, err = srv.Permissions.Insert(file.Id, p).SendNotificationEmails(false).Do()
		if err != nil {
			fmt.Printf("Sharing %s to %s\n", file.Title, accountTo)
			fmt.Printf("shareFile: %s\n", err.Error())
			continue
		}
		break
	}

	return err
}

func share(srv *drive.Service, accountFrom string, accountTo string) error {
	// List all files and folders
	files, err := findAllFilesFrom(srv, accountFrom)
	if err != nil {
		return err
	}
	fmt.Printf("Found: %d files or directories\n", len(files))

	// Progress bar
	bar := pb.New(len(files))
	bar.SetRefreshRate(time.Second)
	bar.Start()

	for _, file := range files {
		bar.Increment()
		err := shareFile(srv, file, accountTo)
		if err != nil {
			return err
		}
	}
	bar.FinishPrint("Done.")

	// Everything is OK
	return nil
}
