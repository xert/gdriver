package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"

	"google.golang.org/api/drive/v2"
)

type task struct {
	ID      string
	Parents []string
}

type result struct {
	ID     string
	Status bool
}

var ReportFile *os.File
var Report *csv.Writer

func init() {
	ReportFile, err := os.OpenFile("./report.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	if err != nil {
		panic(err.Error())
	}
	Report = csv.NewWriter(ReportFile)

	record := []string{
		"S:Title",
		"S:Id",
		"S:MD5",
		"S:Size",
		"D:Title",
		"D:Id",
		"D:MD5",
		"D:Size",
	}
	if err := Report.Write(record); err != nil {
		fmt.Printf("\nERROR writing report: %s \n", err.Error())
	}
}

func migrateFile(srv *drive.Service, t task) error {

	sourceFile, err := srv.Files.Get(t.ID).Do()
	if err != nil {
		fmt.Printf("\nFiles.Get ERROR\n")
		return err
	}

	if !sourceFile.Copyable {
		return fmt.Errorf("%s (%s) is not copyable", sourceFile.Title, t.ID)
	}

	// Construct target
	targetFile := drive.File{Title: sourceFile.Title}
	targetFile.Parents = []*drive.ParentReference{}

	for _, p := range t.Parents {
		targetFile.Parents = append(
			targetFile.Parents,
			&drive.ParentReference{Id: p})
	}

	// Copy file
	resultFile, err := srv.Files.Copy(t.ID, &targetFile).Do()
	if err != nil {
		for _, p := range t.Parents {
			_, e := srv.Files.Get(p).Do()
			if e != nil {
				fmt.Printf("\n###### PARENT ERROR\n")
			}
		}

		return err
	}

	// Remove all permissions
	for _, p := range resultFile.Permissions {
		permission, err := srv.Permissions.Get(resultFile.Id, p.Id).Do()
		if err != nil {
			fmt.Printf("\n\nERROR 101\n\n")
			return err
		}
		if permission.Role != "owner" {
			err := srv.Permissions.Delete(resultFile.Id, p.Id).Do()
			if err != nil {
				fmt.Printf("\n\nERROR 102\n\n")
				return err
			}
		}
	}

	record := []string{
		sourceFile.Title,
		sourceFile.Id,
		sourceFile.Md5Checksum,
		fmt.Sprintf("%d", sourceFile.FileSize),
		resultFile.Title,
		resultFile.Id,
		resultFile.Md5Checksum,
		fmt.Sprintf("%d", resultFile.QuotaBytesUsed),
	}
	if err := Report.Write(record); err != nil {
		fmt.Printf("\nERROR writing report: %s \n", err.Error())
	}

	return nil
}

func worker(id int, srv *drive.Service, tasks <-chan task, results chan<- result) {
	var err error
	for t := range tasks {
		for i := 1; i < 100; i++ {
			if i > 1 {
				fmt.Printf("Worker %d processing job %s (attempt %d)\n", id, t.ID, i)
			}

			err = migrateFile(srv, t)
			if err == nil {
				results <- result{ID: t.ID, Status: true}
				time.Sleep(11 * time.Millisecond * time.Duration(id))
				break
			}
			fmt.Printf("\nE: %s\n", err.Error())
			time.Sleep(111 * time.Millisecond * time.Duration(id))
		}
		if err != nil {
			fmt.Printf("\n==> ERROR: %s\n", err.Error())
			results <- result{ID: t.ID, Status: false}
		}
	}
}

func migrate(srv *drive.Service, accountFrom string) error {
	files, err := ioutil.ReadDir(workDir)
	if err != nil {
		return err
	}

	fmt.Printf("Migrating %d files\n", len(files))

	var tasks []task
	for _, f := range files {
		data, err := ioutil.ReadFile(filepath.Join(workDir, f.Name()))
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		t := task{ID: f.Name(), Parents: []string{}}
		for scanner.Scan() {
			t.Parents = append(t.Parents, scanner.Text())
		}

		tasks = append(tasks, t)
	}

	queue := make(chan task, 1000)
	results := make(chan result, 100)

	// Start some workers
	for w := 1; w <= 5; w++ {
		fmt.Printf("Starting worker %d\n", w)
		go worker(w, srv, queue, results)
	}

	// Send tasks to queue
	go func() {
		for _, t := range tasks {
			queue <- t
			// fmt.Printf("Sent job %s\n", t.ID)
		}
		close(queue)
		fmt.Printf("Sent all jobs\n")
	}()

	// Progress bar
	bar := pb.New(len(files))
	bar.SetRefreshRate(time.Second)
	bar.Start()

	// Receive results
	for range tasks {
		bar.Increment()
		r := <-results
		if r.Status == true {
			// fmt.Printf("SUCCESS job %s\n", r.ID)
			os.Remove(filepath.Join(workDir, r.ID))
		} else {
			fmt.Printf("FAILURE job %s\n", r.ID)
		}
	}

	bar.FinishPrint("Done.")
	empty, _ := isDirEmpty(workDir)
	if empty {
		os.Remove(workDir)
	}

	Report.Flush()
	ReportFile.Close()

	// Everything is OK
	return nil
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// read in ONLY one file
	_, err = f.Readdir(1)

	// and if the file is EOF... well, the dir is empty.
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
