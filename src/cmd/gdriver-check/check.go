package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/cheggaaa/pb"

	"google.golang.org/api/drive/v2"
)

func check(srv *drive.Service, account string) error {

	reportFile, err := os.Open("./report.csv")
	if err != nil {
		return err
	}
	report := csv.NewReader(reportFile)

	records, err := report.ReadAll()
	if err != nil {
		return err
	}

	checksumErrors := 0
	sizeErrors := 0
	nameErrors := 0
	getErrors := 0
	ok := 0

	// Pop first item
	records = records[1:len(records)]

	// Progress bar
	bar := pb.New(len(records))
	bar.SetRefreshRate(time.Second)
	bar.Start()

	for _, record := range records {
		bar.Increment()
		file, err := srv.Files.Get(record[5]).Do()
		if err != nil {
			fmt.Printf("\n\n%s ✖ FILE GET ERROR\n", record[0])
			getErrors++
		} else if file.Title != record[0] {
			fmt.Printf("\n\n%s ✖ NAME ERROR %s != %s\n", record[0], file.Title, record[0])
			nameErrors++
		} else if file.Md5Checksum != record[2] {
			fmt.Printf("\n\n%s ✖ MD5 MISMATCH %s != %d\n", record[0], record[2], file.Md5Checksum)
			checksumErrors++
		} else if fmt.Sprintf("%d", file.QuotaBytesUsed) != record[3] {
			fmt.Printf("\n\n%s ✖ SIZE MISMATCH %s != %d\n", record[0], record[3], file.QuotaBytesUsed)
			sizeErrors++
		} else {
			ok++
		}
	}

	bar.FinishPrint("Done.")

	fmt.Printf("RESULTS:\n%d Checksum errors\n%d Size errors\n%d Name errors\n%d Fetch errors\n%d OK", checksumErrors, sizeErrors, nameErrors, getErrors, ok)

	// Everything is OK
	return nil
}
