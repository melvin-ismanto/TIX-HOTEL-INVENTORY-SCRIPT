package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

func main() {
	basePath := "/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/"
	inputFile := basePath + "integrator-framework-new-list.csv"
	outputFile := basePath + "IF_new_blacklist_rollout_v2.csv"

	// Open input CSV
	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Read header
	header, err := reader.Read()
	if err != nil {
		fmt.Println("Error reading header:", err)
		return
	}

	// Validate header
	expected := []string{"hotelId", "vendor", "roomId"}
	for i, h := range expected {
		if header[i] != h {
			fmt.Printf("Unexpected header at column %d: got %s, want %s\n", i, header[i], h)
			return
		}
	}

	// Track hotel_vendor_id seen for HOTEL blacklist
	seenHotelVendorIDs := make(map[string]bool)
	allVendors := make(map[string]struct{})

	// Prepare output rows
	var output [][]string
	output = append(output, []string{"type", "vendor", "value"}) // header

	// Read and process rows
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}

		vendor := strings.ToUpper(row[1])
		roomSupplierID := row[2]
		hotelVendorID := row[0]

		if vendor == "" || roomSupplierID == "" || hotelVendorID == "" {
			continue
		}

		if vendor == "TIKET" {
			continue
		}

		allVendors[vendor] = struct{}{}

		// If hotelVendorID not seen, add HOTEL blacklist row
		if !seenHotelVendorIDs[vendor+hotelVendorID] {
			output = append(output, []string{
				"FILTERING_BLACKLIST:HOTEL",
				vendor,
				hotelVendorID,
			})
			seenHotelVendorIDs[vendor+hotelVendorID] = true
		}

		// Always add ROOM blacklist row
		output = append(output, []string{
			"FILTERING_BLACKLIST:ROOM",
			vendor,
			fmt.Sprintf("%s:%s", hotelVendorID, roomSupplierID),
		})
	}

	// Write output to new CSV file
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	writer.WriteAll(output)
	writer.Flush()

	if err := writer.Error(); err != nil {
		fmt.Println("CSV write error:", err)
	}

	// append all vendors
	allVendorString := ""
	for key := range allVendors {
		allVendorString += "," + key
	}
	if allVendorString != "" {
		allVendorString = allVendorString[1:] // remove the first comma
	}
	fmt.Println("all vendors : " + allVendorString)
}
