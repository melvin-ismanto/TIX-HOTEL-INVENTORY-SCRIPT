package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Whitelisted struct {
	Mode   string   `json:"mode"`
	Values []string `json:"values"`
}

type RolloutConfig struct {
	Toggle            string              `json:"toggle"`
	Whitelisted       Whitelisted         `json:"whitelisted"`
	BlacklistedRoomId map[string][]string `json:"blacklistedRoomId"`
}

// Load original rollout JSON as map[vendor]RolloutConfig
func readOriginalRolloutMap(filename string) (map[string]RolloutConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result map[string]RolloutConfig
	err = json.NewDecoder(file).Decode(&result)
	return result, err
}

// Read CSV and group into map[vendor]RolloutConfig
func readCSVAndGroupByVendor(filename string) (map[string]RolloutConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	headerIndex := map[string]int{}
	for i, h := range headers {
		headerIndex[h] = i
	}

	result := make(map[string]RolloutConfig)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		vendor := record[headerIndex["vendor"]]
		hotelVendorID := record[headerIndex["hotel_vendor_id"]]
		roomSupplierID := record[headerIndex["room_supplier_id"]]

		rollout := result[vendor]
		if rollout.BlacklistedRoomId == nil {
			rollout.BlacklistedRoomId = make(map[string][]string)
			rollout.Toggle = "ACTIVE"
			rollout.Whitelisted = Whitelisted{Mode: "ALL", Values: []string{}}
		}

		rollout.BlacklistedRoomId[hotelVendorID] = append(
			rollout.BlacklistedRoomId[hotelVendorID],
			roomSupplierID,
		)

		result[vendor] = rollout
	}

	return result, nil
}

// Merge each vendor-specific rollout
func mergeRolloutMaps(original, new map[string]RolloutConfig) map[string]RolloutConfig {
	merged := make(map[string]RolloutConfig)

	// Start with original
	for vendor, orig := range original {
		merged[vendor] = orig
	}

	// Merge with new
	for vendor, newRollout := range new {
		mergedRollout := merged[vendor]
		if mergedRollout.BlacklistedRoomId == nil {
			mergedRollout.BlacklistedRoomId = make(map[string][]string)
			mergedRollout.Toggle = "ACTIVE"
			mergedRollout.Whitelisted = Whitelisted{Mode: "ALL", Values: []string{}}
		}
		for hotelID, newRooms := range newRollout.BlacklistedRoomId {
			mergedRooms := append(mergedRollout.BlacklistedRoomId[hotelID], newRooms...)
			mergedRollout.BlacklistedRoomId[hotelID] = unique(mergedRooms)
		}
		merged[vendor] = mergedRollout
	}

	return merged
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
	basePath := "/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/"
	jsonFile := basePath + "original_rollout.json"
	csvFile := basePath + "20250620 New All Ordered Rooms 15.37.24.csv"
	outputFile := basePath + "multi_vendor_merged_rollout.json"

	originalMap, err := readOriginalRolloutMap(jsonFile)
	if err != nil {
		fmt.Println("Error reading original rollout JSON:", err)
		return
	}

	newMap, err := readCSVAndGroupByVendor(csvFile)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	mergedMap := mergeRolloutMaps(originalMap, newMap)

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(mergedMap); err != nil {
		fmt.Println("Error writing merged JSON:", err)
		return
	}

	fmt.Printf("Merged rollout map written to %s\n", outputFile)
}
