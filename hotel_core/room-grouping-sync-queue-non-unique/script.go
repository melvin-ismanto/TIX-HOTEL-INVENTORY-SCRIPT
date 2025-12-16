package room_grouping_sync_queue_non_unique

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// RawData represents the JSON structure in the raw field
type RawData struct {
	Raw string `json:"raw"`
}

// HotelGrouping represents a unique hotelId and groupingKey pair
type HotelGrouping struct {
	HotelID     string
	GroupingKey string
}

func RunScript() {
	// Read CSV file
	basePath := "/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/"
	file, err := os.Open(basePath + "roomGroupingImageQueueAck non unique.csv")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Set quote character
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV: %v\n", err)
		return
	}

	fmt.Printf("Total rows in original file: %d\n", len(records))

	// Map to store unique hotelId-groupingKey pairs
	uniquePairs := make(map[string]HotelGrouping)

	// Improved regex patterns to extract hotelId and groupingKey
	// hotelId is UUID format
	hotelIDPattern := regexp.MustCompile(`"hotelId"\s*:\s*"([a-fA-F0-9\-]+)"`)
	// groupingKey can contain letters, numbers, hyphens, and underscores (including "cetus-")
	groupingKeyPattern := regexp.MustCompile(`"groupingKey"\s*:\s*"([^"]+)"`)

	// Skip header if exists and process each row
	startIndex := 0
	if len(records) > 0 && strings.Contains(records[0][0], "@timestamp") {
		startIndex = 1 // Skip header
	}

	for i := startIndex; i < len(records); i++ {
		if len(records[i]) < 3 {
			continue // Skip rows without enough columns
		}

		rawField := records[i][2] // raw field is the third column

		// Extract hotelId and groupingKey
		hotelID, groupingKey := extractIDsFromRaw(rawField, hotelIDPattern, groupingKeyPattern)

		if hotelID != "" && groupingKey != "" {
			// Create a unique key for the map
			key := hotelID + "|" + groupingKey
			uniquePairs[key] = HotelGrouping{
				HotelID:     hotelID,
				GroupingKey: groupingKey,
			}
		}
	}

	fmt.Printf("Unique hotelId-groupingKey pairs found: %d\n\n", len(uniquePairs))

	// Display unique pairs
	fmt.Println("Unique hotelId and groupingKey pairs:")
	for _, pair := range uniquePairs {
		fmt.Printf("%-40s %s\n", pair.HotelID, pair.GroupingKey)
	}

	// Save to CSV
	err = saveToCSV(uniquePairs, "unique_hotelId_groupingKey_pairs.csv")
	if err != nil {
		fmt.Printf("Error saving to CSV: %v\n", err)
		return
	}

	fmt.Printf("\nSaved unique pairs to 'unique_hotelId_groupingKey_pairs.csv'\n")
}

// extractIDsFromRaw extracts hotelId and groupingKey from the raw JSON string
func extractIDsFromRaw(rawStr string, hotelIDPattern, groupingKeyPattern *regexp.Regexp) (string, string) {
	// Parse the raw JSON string
	var rawData RawData
	err := json.Unmarshal([]byte(rawStr), &rawData)
	if err != nil {
		// Try to fix the JSON if it has escaped quotes
		unescaped := strings.ReplaceAll(rawStr, `\"`, `"`)
		err = json.Unmarshal([]byte(unescaped), &rawData)
		if err != nil {
			return "", ""
		}
	}

	// Extract hotelId
	hotelIDMatch := hotelIDPattern.FindStringSubmatch(rawData.Raw)
	hotelID := ""
	if len(hotelIDMatch) > 1 {
		hotelID = hotelIDMatch[1]
	}

	// Extract groupingKey
	groupingKeyMatch := groupingKeyPattern.FindStringSubmatch(rawData.Raw)
	groupingKey := ""
	if len(groupingKeyMatch) > 1 {
		groupingKey = groupingKeyMatch[1]
	}

	return hotelID, groupingKey
}

// saveToCSV saves the unique pairs to a CSV file
func saveToCSV(pairs map[string]HotelGrouping, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"hotelId", "groupingKey"})
	if err != nil {
		return err
	}

	// Write data
	for _, pair := range pairs {
		err = writer.Write([]string{pair.HotelID, pair.GroupingKey})
		if err != nil {
			return err
		}
	}

	return nil
}
