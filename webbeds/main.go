package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type WebbedsWhitelisted struct {
	Mode   string   `json:"mode"`
	Values []string `json:"values"`
}

type WebbedsRollout struct {
	Toggle            string              `json:"toggle"`
	Whitelisted       WebbedsWhitelisted  `json:"whitelisted"`
	BlacklistedRoomId map[string][]string `json:"blacklistedRoomId"`
}

func readOriginalRollout(filename string) (WebbedsRollout, error) {
	var rollout WebbedsRollout
	file, err := os.Open(filename)
	if err != nil {
		return rollout, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&rollout)
	return rollout, err
}

func readCSVAndConstructRollout(filename string, vendorFilter string) (WebbedsRollout, error) {
	file, err := os.Open(filename)
	if err != nil {
		return WebbedsRollout{}, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return WebbedsRollout{}, err
	}

	headerIndex := map[string]int{}
	for i, h := range headers {
		headerIndex[h] = i
	}

	blacklisted := make(map[string][]string)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return WebbedsRollout{}, err
		}

		if record[headerIndex["vendor"]] != vendorFilter {
			continue
		}

		hotelVendorID := record[headerIndex["property_supplier_id"]]
		roomSupplierID := record[headerIndex["room_supplier_id"]]

		if hotelVendorID == "" || roomSupplierID == "" {
			continue
		}

		blacklisted[hotelVendorID] = append(blacklisted[hotelVendorID], roomSupplierID)
	}

	return WebbedsRollout{
		Toggle: "ACTIVE",
		Whitelisted: WebbedsWhitelisted{
			Mode:   "ALL",
			Values: []string{},
		},
		BlacklistedRoomId: blacklisted,
	}, nil
}

func mergeRollouts(original, new WebbedsRollout) WebbedsRollout {
	for key, newList := range new.BlacklistedRoomId {
		origList := original.BlacklistedRoomId[key]
		merged := append(origList, newList...)
		original.BlacklistedRoomId[key] = unique(merged)
	}
	return original
}

func unique(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range input {
		if !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	return result
}

func main() {
	// File paths
	basePath := "/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/"
	jsonFile := basePath + "webbeds_original_rollout.json"
	csvFile := basePath + "new_webbeds.csv"
	vendorFilter := "webbeds"
	outputFile := basePath + strings.ToLower(vendorFilter) + "_merged_rollout_v2.json"

	original, err := readOriginalRollout(jsonFile)
	if err != nil {
		fmt.Println("Error reading JSON:", err)
		return
	}

	newRollout, err := readCSVAndConstructRollout(csvFile, vendorFilter)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	merged := mergeRollouts(original, newRollout)

	// Write to file
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(merged); err != nil {
		fmt.Println("Error writing JSON to file:", err)
		return
	}

	fmt.Printf("Merged rollout written to %s\n", outputFile)
}
