package master_query

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	csvFilePath = ""
	apiTemplate = "http://hotel-core-be-svc.prod-hotel-cluster.tiket.com/tix-hotel-core/master/execute-query?queryUpdateType=SET&collectionName=room_grouping_properties&id=%s&queryObjectId=true"
	jsonBody    = `{
	"cetusVersion": "CETUS_V2"
}`
)

func removeZeroWidthSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\uFEFF' {
			return -1 // remove the rune
		}
		return r
	}, s)
}

func UpdateCetusVersion() {
	file, err := os.Open(csvFilePath)
	if err != nil {
		panic(fmt.Errorf("failed to open CSV file: %w", err))
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read() // Read the header row
	if err != nil {
		panic(fmt.Errorf("failed to read CSV header: %w", err))
	}

	// Find the _id column index
	idIndex := -1
	for i, h := range headers {
		if removeZeroWidthSpaces(h) == "_id" {
			idIndex = i
			break
		}
	}
	if idIndex == -1 {
		panic("missing _id column in CSV")
	}

	// Read and process each row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("error reading record: %v\n", err)
			continue
		}

		id := removeZeroWidthSpaces(record[idIndex])
		url := fmt.Sprintf(apiTemplate, id)

		req, err := http.NewRequest("POST", url, strings.NewReader(jsonBody))
		if err != nil {
			fmt.Printf("failed to build request for ID %s: %v\n", id, err)
			continue
		}

		req.Header.Set("storeId", "TIKETCOM")
		req.Header.Set("channelId", "WEB")
		req.Header.Set("requestId", "23123123")
		req.Header.Set("serviceId", "SWAGGER")
		req.Header.Set("username", "melvin.ismanto")
		req.Header.Set("lang", "en")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("request failed for ID %s: %v\n", id, err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("ID: %s | Status: %d\n", id, resp.StatusCode)
	}
}
