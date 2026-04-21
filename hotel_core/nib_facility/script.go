package nib_facility

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	basePath = "/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/"
	csvFile  = "NIB Data 31_03_2026 11_50 - HashID data.csv"
	baseURL  = "http://hotel-core-be-svc.prod-hotel-cluster.tiket.com/tix-hotel-core/hotel"
)

func RunScript() {
	file, err := os.Open(basePath + csvFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		fmt.Printf("Error reading header: %v\n", err)
		return
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	hashIDIdx, ok1 := colIndex["property_hash_id"]
	supplierIDIdx, ok2 := colIndex["supplier_property_id"]
	if !ok1 || !ok2 {
		fmt.Println("Error: required columns not found in CSV")
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}

	rowNum := 0
	successCount := 0
	failCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading row: %v\n", err)
			continue
		}

		rowNum++
		propertyHashID := strings.TrimSpace(record[hashIDIdx])
		supplierPropertyID := strings.TrimSpace(record[supplierIDIdx])

		if propertyHashID == "" || supplierPropertyID == "" {
			fmt.Printf("[Row %d] SKIP - empty property_hash_id or supplier_property_id\n", rowNum)
			failCount++
			continue
		}

		err = patchFacility(client, propertyHashID, supplierPropertyID)
		if err != nil {
			fmt.Printf("[Row %d] FAIL - hash_id=%s supplier_id=%s err=%v\n", rowNum, propertyHashID, supplierPropertyID, err)
			failCount++
			continue
		}

		fmt.Printf("[Row %d] OK - hash_id=%s supplier_id=%s\n", rowNum, propertyHashID, supplierPropertyID)
		successCount++
	}

	fmt.Printf("\nDone. total=%d success=%d fail=%d\n", rowNum, successCount, failCount)
}

func patchFacility(client *http.Client, propertyHashID, supplierPropertyID string) error {
	url := fmt.Sprintf("%s/%s/facility", baseURL, propertyHashID)

	body := fmt.Sprintf(`{
  "facilityIds": {
    "source": "TIKET",
    "sourceId": "%s",
    "value": [],
    "isUsingPreferredVendor": false
  }
}`, supplierPropertyID)

	req, err := http.NewRequest(http.MethodPatch, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("storeId", "TIKETCOM")
	req.Header.Set("channelId", "WEB")
	req.Header.Set("requestId", "on-call-AC-45530")
	req.Header.Set("serviceId", "LOGIN")
	req.Header.Set("X-Account-Id", "0")
	req.Header.Set("username", "on-call-AC-45530")
	req.Header.Set("lang", "en")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}
