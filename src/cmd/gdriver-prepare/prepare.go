package main

import (
	"bytes"
	"fmt"
	"gdrive"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"

	"google.golang.org/api/drive/v2"
)

func findAllFilesFrom(srv *drive.Service, accountFrom string) ([]*drive.File, error) {
	var f []*drive.File
	pageToken := ""
	for {
		q := srv.Files.List().Q("'" + accountFrom + "' in owners").MaxResults(1000)
		// If we have a pageToken set, apply it to the query
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}
		r, err := q.Do()
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

func createFolder(srv *drive.Service, folder *drive.File, folderMap map[string]string, rootFolder *drive.File) (*drive.File, error) {
	newFolder := &drive.File{Title: folder.Title, MimeType: gdrive.FolderMIME}

	newFolder.Parents = []*drive.ParentReference{}
	underRoot := false

	if len(folder.Parents) == 0 {
		underRoot = true
	}

	for _, p := range folder.Parents {
		if p.IsRoot {
			underRoot = true
			continue
		}

		if _, ok := folderMap[p.Id]; !ok {
			//fmt.Printf(" no parent %s\n", p.Id)
			// Parent folder does not exist
			return nil, nil
		}

		parent := &drive.ParentReference{Id: folderMap[p.Id]}
		newFolder.Parents = append(newFolder.Parents, parent)
	}

	if underRoot {
		parent := &drive.ParentReference{Id: rootFolder.Id}
		newFolder.Parents = append(newFolder.Parents, parent)
	}

	r, err := srv.Files.Insert(newFolder).Do()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func prepare(srv *drive.Service, accountFrom string) error {
	// Create working directory
	err := os.Mkdir(workDir, 0770)
	if err != nil {
		return err
	}

	// List all files and folders
	files, err := findAllFilesFrom(srv, accountFrom)
	if err != nil {
		return err
	}
	fmt.Printf("Found: %d files or directories\n", len(files))

	// Create new root folder
	rf := &drive.File{Title: "MIGRACE", MimeType: gdrive.FolderMIME}
	fmt.Printf("Creating root folder %s", rf.Title)
	rootFolder, err := srv.Files.Insert(rf).Do()
	if err != nil {
		fmt.Println(" FAILED")
		return err
	}
	fmt.Printf(" SUCCESS (%s)\n", rootFolder.Id)

	// Find all folders
	oldFolders := map[string]*drive.File{}
	var folders []string
	for _, f := range files {
		if f.MimeType == gdrive.FolderMIME { // folder
			oldFolders[f.Id] = f
			folders = append(folders, f.Id)
		}
	}

	// Iterate all folders and create new
	fmt.Printf("\nCreating %d folders\n", len(oldFolders))
	folderMap := map[string]string{}
	newFolders := map[string]*drive.File{}
	var f *drive.File
	pass := 0
	bar := pb.New(len(oldFolders))
	bar.SetRefreshRate(time.Second)
	bar.Start()
	for len(newFolders) != len(oldFolders) {
		pass++
		fmt.Printf("== PASS %d ===================\n", pass)
		if pass > len(oldFolders)*2 {
			return fmt.Errorf("Too many iterations") // handbrake
		}
		for id, folder := range oldFolders {
			if _, ok := folderMap[id]; ok {
				continue // Folder exists
			}

			// Five tries to create folder
			i := 0
			for i <= 5 {
				i++
				// fmt.Printf("Creating folder %s (attempt %d)", folder.Title, i)
				f, err = createFolder(srv, folder, folderMap, rootFolder)
				if err == nil {
					if f != nil { // something was created
						// fmt.Printf(" CREATED (%s)\n", f.Id)
						newFolders[f.Id] = f
						folderMap[id] = f.Id
						bar.Increment()
					} else {
						// fmt.Print(" NO PARENTS\n")
					}
					break
				} else {
					fmt.Printf("E: %s\n", err.Error())
				}
			}

			// Can't create folder - quit
			if err != nil {
				return err
			}
		}
	}
	bar.Finish()
	fmt.Printf("\n%d folders created in %d pass(es)\n", len(newFolders), pass)

	// Work files
	fmt.Printf("\nDumping %d files to workdir\n", len(files))
	bar = pb.StartNew(len(files))
	for _, f := range files {
		bar.Increment()
		if f.MimeType == gdrive.FolderMIME { // folder
			continue
		}

		parents := new(bytes.Buffer)
		for _, p := range f.Parents {
			fmt.Fprintf(parents, "%s\n", folderMap[p.Id])
		}
		err := ioutil.WriteFile(filepath.Join(workDir, f.Id), parents.Bytes(), 0660)
		if err != nil {
			return err
		}
	}
	bar.FinishPrint("Prepare finished.")

	// Everything is OK
	return nil
}
