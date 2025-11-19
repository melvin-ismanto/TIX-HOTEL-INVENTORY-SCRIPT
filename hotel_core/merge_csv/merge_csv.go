package merge_csv

import (
	"encoding/csv"
	"log"
	"os"
	"strings"
)

func MergeCsv() {
	csv1 := readCSV("/Users/melvinismanto/Desktop/tmp/export_no_grouping_logic.csv")
	csv2 := readCSV("/Users/melvinismanto/Desktop/tmp/diff.csv")

	// Convert to sets for fast lookup
	set1 := make(map[string]bool)
	for _, row := range csv1[1:] { // skip header
		id := strings.TrimSpace(row[0])
		if id != "" {
			set1[id] = true
		}
	}

	// Compute difference: values in csv2 not in csv1
	var diff [][]string
	diff = append(diff, []string{"hotelId_not_in_csv1"}) // header

	seen := make(map[string]bool) // avoid duplicates
	for _, row := range csv2[1:] {
		id := strings.TrimSpace(row[0])
		if id == "" {
			continue
		}
		if !set1[id] && !seen[id] {
			diff = append(diff, []string{id})
			seen[id] = true
		}
	}

	writeCSV("/Users/melvinismanto/Desktop/tmp/"+"result.csv", diff)
	log.Println("Successfully result diff.csv")
}

func readCSV(filename string) [][]string {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open %s: %v", filename, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatalf("failed to read %s: %v", filename, err)
	}
	return records
}

func writeCSV(filename string, data [][]string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("failed to create %s: %v", filename, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, row := range data {
		if err := w.Write(row); err != nil {
			log.Fatalf("failed to write row: %v", err)
		}
	}
}
