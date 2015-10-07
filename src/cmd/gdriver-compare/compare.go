package main

import (
	"fmt"
	"gdrive"

	"google.golang.org/api/drive/v2"
)

type file struct {
	ID          string
	Title       string
	MimeType    string
	MD5Checksum string
}

type folder struct {
	ID      string
	Title   string
	Folders []folder
	Files   []file
}

type node struct {
	ID          string
	Title       string
	MimeType    string
	MD5Checksum string
	Path        string
}

func gdriveFolderExists(srv *drive.Service, id string) (bool, error) {
	var err error
	var file *drive.File

	for i := 0; i < 100; i++ {
		file, err = srv.Files.Get(id).Do()
		if err == nil {
			break
		}
	}

	if err != nil {
		return false, err
	}

	if file.MimeType != gdrive.FolderMIME {
		return false, nil
	}

	return true, nil
}

func gdriveChildren(srv *drive.Service, folderID string) ([]folder, []file, error) {
	var err error
	var cs []*drive.ChildReference
	var q *drive.ChildrenListCall

	pageToken := ""
	for {
		for i := 0; i < 100; i++ {
			q = srv.Children.List(folderID)
			if err == nil {
				break
			}
		}
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}
		r, err := q.Do()
		if err != nil {
			return nil, nil, err
		}
		cs = append(cs, r.Items...)
		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}

	folders := []folder{}
	files := []file{}
	var node *drive.File

	for _, c := range cs {
		for i := 0; i < 100; i++ {
			node, err = srv.Files.Get(c.Id).Do()
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, nil, err
		}

		if node.MimeType == gdrive.FolderMIME {
			folders = append(folders, folder{ID: node.Id, Title: node.Title})
		} else {
			files = append(files, file{ID: node.Id, Title: node.Title, MimeType: node.MimeType, MD5Checksum: node.Md5Checksum})
		}
	}

	return folders, files, nil
}

func gdriveTree(srv *drive.Service, rootID string, f *folder) error {
	var err error
	var folders []folder
	var files []file

	fmt.Print(".")
	for i := 0; i < 100; i++ {
		folders, files, err = gdriveChildren(srv, rootID)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	for i, n := range folders {
		for j := 0; j < 100; j++ {
			err = gdriveTree(srv, n.ID, &folders[i])
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
	}

	f.Files = append(f.Files, files...)
	f.Folders = append(f.Folders, folders...)

	return nil
}

func generateTree(srv *drive.Service, rootID string) (*folder, error) {
	var tree folder

	exists, err := gdriveFolderExists(srv, rootID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Folder %s does not exist or is not a folder", rootID)
	}

	err = gdriveTree(srv, rootID, &tree)
	if err != nil {
		return nil, err
	}

	return &tree, nil
}

func flattenTree(prefix string, t *folder) []node {
	var nodes []node

	if t == nil {
		return nodes
	}

	for _, n := range t.Files {
		nodes = append(nodes, node{
			ID:          n.ID,
			Title:       n.Title,
			MimeType:    n.MimeType,
			MD5Checksum: n.MD5Checksum,
			Path:        fmt.Sprintf("%s%s", prefix, n.Title),
		})
	}

	for _, n := range t.Folders {
		node := node{
			ID:          n.ID,
			Title:       n.Title + "/",
			MimeType:    gdrive.FolderMIME,
			MD5Checksum: "0",
			Path:        fmt.Sprintf("%s%s/", prefix, n.Title),
		}
		nodes = append(nodes, node)

		sub := flattenTree(node.Path, &n)
		if sub != nil {
			nodes = append(nodes, sub...)
		}

	}

	return nodes
}

func compare(srv *drive.Service, id1, id2 string) error {
	tree1, err := generateTree(srv, id1)
	if err != nil {
		return err
	}

	flat1 := flattenTree("/", tree1)

	files := 0
	folders := 0
	for _, n := range flat1 {
		if n.MimeType == gdrive.FolderMIME {
			folders++
		} else {
			files++
		}
	}

	fmt.Printf("A: Found %d files and %d folders\n", files, folders)

	tree2, err := generateTree(srv, id2)
	if err != nil {
		return err
	}

	flat2 := flattenTree("/", tree2)

	files = 0
	folders = 0
	for _, n := range flat2 {
		if n.MimeType == gdrive.FolderMIME {
			folders++
		} else {
			files++
		}
	}

	fmt.Printf("B: Found %d files and %d folders\n", files, folders)

	return nil
}
