package master_query

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Room struct {
	ID        string `bson:"_id"`
	IsDeleted int    `bson:"isDeleted"`
	IsActive  int    `bson:"isActive"`
	HotelID   string `bson:"hotelId"`
	Vendor    Vendor `bson:"vendor"`
}

type Vendor struct {
	Name                  string `bson:"name"`
	VendorHotelExternalID string `bson:"vendorHotelExternalId"`
	VendorExternalID      string `bson:"vendorExternalId"`
}

type RoomGrouping struct {
	ID          string `bson:"_id"`
	GroupingKey string `bson:"groupingKey"`
}

type CSVRecord struct {
	PKFAREHotelID     string
	PKFARERoomID      string
	PKFARERoomName    string
	ExpediaHotelID    string
	ExpediaRoomID     string
	ExpediaRoomName   string
	PKFAREHotelHashID string
	PKFARERoomMongoID string
	RoomHashID        string
	// New fields for output
	GroupingKey string
	Status      string
	Notes       string
	// Internal room MongoDB ID from database query
	InternalRoomID string
	HashID         string
}

func FindRoomGroupingKey() {
	log.Println("Starting FindRoomGroupingKey process...")

	// MongoDB connection
	clientOpts := options.Client().ApplyURI("mongodb://mfaji:Vxu4pesbH5@10.145.128.146:27017/admin")
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.TODO())

	db := client.Database("hotel_core")
	roomCollection := db.Collection("room")
	roomGroupingCollection := db.Collection("room_grouping")

	// Read CSV file
	inputFile, err := os.Open("resource/PKFARE Mapping - room cross mapping.csv")
	if err != nil {
		log.Fatalf("Failed to open input CSV file: %v", err)
	}
	defer inputFile.Close()

	csvReader := csv.NewReader(inputFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v", err)
	}

	if len(records) == 0 {
		log.Fatal("CSV file is empty")
	}

	log.Printf("Processing %d records...", len(records)-1)

	// Step 1: Pre-process CSV records and group by vendorHotelExternalId
	type RecordKey struct {
		HotelID string
		RoomID  string
	}

	var allCSVRecords []CSVRecord
	recordsByHotel := make(map[string][]int) // hotelID -> indices in allCSVRecords
	recordKeys := make([]RecordKey, 0)
	var results []CSVRecord

	// Collect and pre-process all valid records
	for i, record := range records[1:] {
		if len(record) < 9 {
			log.Printf("Row %d has insufficient columns, skipping", i+2)
			continue
		}

		// Clean up Expedia room ID (remove quotes and commas)
		cleanedRoomID := strings.ReplaceAll(record[4], `"`, "")
		cleanedRoomID = strings.ReplaceAll(cleanedRoomID, `,`, "")

		csvRecord := CSVRecord{
			PKFAREHotelID:     record[0],
			PKFARERoomID:      record[1],
			PKFARERoomName:    record[2],
			ExpediaHotelID:    record[3],
			ExpediaRoomID:     cleanedRoomID,
			ExpediaRoomName:   record[5],
			PKFAREHotelHashID: record[6],
			PKFARERoomMongoID: record[7],
			RoomHashID:        record[8],
		}

		// Skip rows with empty Expedia data
		if csvRecord.ExpediaHotelID == "" || csvRecord.ExpediaRoomID == "" {
			csvRecord.Status = "FAILED"
			csvRecord.Notes = "Input: Missing Expedia hotel ID or room ID"
			results = append(results, csvRecord)
			continue
		}

		// Store record for batch processing
		allCSVRecords = append(allCSVRecords, csvRecord)

		// Group by hotel for batch processing
		hotelIndices := recordsByHotel[csvRecord.ExpediaHotelID]
		recordsByHotel[csvRecord.ExpediaHotelID] = append(hotelIndices, len(allCSVRecords)-1)

		recordKeys = append(recordKeys, RecordKey{
			HotelID: csvRecord.ExpediaHotelID,
			RoomID:  cleanedRoomID,
		})
	}

	log.Printf("Pre-processing complete. Found %d unique hotels to process", len(recordsByHotel))

	// Step 2: Batch query room collection by hotel
	roomMap := make(map[RecordKey]*Room)

	for hotelID, indices := range recordsByHotel {
		// Collect all room IDs for this hotel
		var roomIDs []string
		for _, idx := range indices {
			cleanedRoomID := strings.ReplaceAll(allCSVRecords[idx].ExpediaRoomID, `"`, "")
			cleanedRoomID = strings.ReplaceAll(cleanedRoomID, `,`, "")
			roomIDs = append(roomIDs, cleanedRoomID)
		}

		// Batch query for all rooms of this hotel
		roomFilter := bson.M{
			"vendor.name":                  "EXPEDIA_RAPID",
			"vendor.vendorHotelExternalId": hotelID,
			"vendor.vendorExternalId":      bson.M{"$in": roomIDs},
		}

		projection := bson.M{
			"isDeleted": 1,
			"isActive":  1,
			"hotelId":   1,
			"vendor":    1,
		}

		cursor, err := roomCollection.Find(context.TODO(), roomFilter, options.Find().SetProjection(projection))
		if err != nil {
			log.Printf("Error querying rooms for hotel %s: %v", hotelID, err)
			continue
		}

		var rooms []Room
		if err := cursor.All(context.TODO(), &rooms); err != nil {
			log.Printf("Error decoding rooms for hotel %s: %v", hotelID, err)
			cursor.Close(context.TODO())
			continue
		}
		cursor.Close(context.TODO())

		// Map rooms by their key
		for _, room := range rooms {
			key := RecordKey{
				HotelID: room.Vendor.VendorHotelExternalID,
				RoomID:  room.Vendor.VendorExternalID,
			}
			roomMap[key] = &room
		}
	}

	log.Printf("Room batch query complete. Found %d rooms", len(roomMap))

	// Step 3: Collect valid room IDs grouped by hotelId for room_grouping batch query
	groupingQueries := make(map[string][]string) // hotelId -> roomIds
	recordToRoom := make(map[int]*Room)          // record index -> room

	for i, record := range allCSVRecords {
		if record.Status == "FAILED" {
			continue // Skip already failed records
		}

		cleanedRoomID := strings.ReplaceAll(record.ExpediaRoomID, `"`, "")
		cleanedRoomID = strings.ReplaceAll(cleanedRoomID, `,`, "")

		key := RecordKey{
			HotelID: record.ExpediaHotelID,
			RoomID:  cleanedRoomID,
		}

		room, found := roomMap[key]
		if !found {
			allCSVRecords[i].Status = "FAILED"
			allCSVRecords[i].Notes = "Query: Expedia Room not found in database"
			continue
		}

		// Check if room is deleted or inactive
		if room.IsDeleted == 1 {
			allCSVRecords[i].Status = "FAILED"
			allCSVRecords[i].Notes = "Query: Room is deleted (isDeleted=1)"
			continue
		}

		if room.IsActive == 0 {
			allCSVRecords[i].Status = "FAILED"
			allCSVRecords[i].Notes = "Query: Room is inactive (isActive=0)"
			continue
		}

		// Store valid room for grouping query
		recordToRoom[i] = room
		roomIDs := groupingQueries[room.HotelID]
		groupingQueries[room.HotelID] = append(roomIDs, room.ID)
	}

	log.Printf("Room validation complete. %d hotels need grouping queries", len(groupingQueries))

	// Step 4: Batch query room_grouping collection by hotelId
	groupingMap := make(map[string]map[string]string) // hotelId -> roomId -> groupingKey

	for hotelID, roomIDs := range groupingQueries {
		groupingFilter := bson.M{
			"hotelId":         hotelID,
			"roomList.roomId": bson.M{"$in": roomIDs},
		}

		groupingProjection := bson.M{
			"groupingKey":     1,
			"roomList.roomId": 1,
		}

		cursor, err := roomGroupingCollection.Find(context.TODO(), groupingFilter, options.Find().SetProjection(groupingProjection))
		if err != nil {
			log.Printf("Error querying room_grouping for hotel %s: %v", hotelID, err)
			continue
		}

		var groupings []struct {
			GroupingKey string `bson:"groupingKey"`
			RoomList    []struct {
				RoomID string `bson:"roomId"`
			} `bson:"roomList"`
		}

		if err := cursor.All(context.TODO(), &groupings); err != nil {
			log.Printf("Error decoding room_grouping for hotel %s: %v", hotelID, err)
			cursor.Close(context.TODO())
			continue
		}
		cursor.Close(context.TODO())

		// Build mapping
		hotelGroupings := make(map[string]string)
		for _, grouping := range groupings {
			if grouping.GroupingKey != "" {
				for _, roomItem := range grouping.RoomList {
					hotelGroupings[roomItem.RoomID] = grouping.GroupingKey
				}
			}
		}
		groupingMap[hotelID] = hotelGroupings
	}

	log.Printf("Room grouping batch query complete")

	// Step 5: Process final results
	for i, record := range allCSVRecords {
		if record.Status == "FAILED" {
			results = append(results, record)
			continue
		}

		room, hasRoom := recordToRoom[i]
		if !hasRoom {
			results = append(results, record)
			continue
		}

		hotelGroupings, hasHotel := groupingMap[room.HotelID]
		if !hasHotel {
			allCSVRecords[i].Status = "FAILED"
			allCSVRecords[i].Notes = "Query: Room GROUPING not found"
			results = append(results, allCSVRecords[i])
			continue
		}

		groupingKey, hasGrouping := hotelGroupings[room.ID]
		if !hasGrouping || groupingKey == "" {
			allCSVRecords[i].Status = "FAILED"
			allCSVRecords[i].Notes = "Query: Room GROUPING not found"
			results = append(results, allCSVRecords[i])
			continue
		}

		// Success case
		allCSVRecords[i].Status = "SUCCESS"
		allCSVRecords[i].GroupingKey = groupingKey
		allCSVRecords[i].Notes = "Successfully found grouping key"
		allCSVRecords[i].InternalRoomID = room.ID
		allCSVRecords[i].HashID = room.HotelID
		results = append(results, allCSVRecords[i])
	}

	// Step 4: Export to CSV
	outputFile, err := os.Create("resource/PKFARE_Mapping_with_grouping_key_output.csv")
	if err != nil {
		log.Fatalf("Failed to create output CSV file: %v", err)
	}
	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"PKFARE_hotel_id", "PKFARE_room_id", "PKFARE_room_name",
		"Expedia_hotel_id", "Expedia_room_id", "Expedia_room_name",
		"PKFARE HOTEL HASH ID", "PKFARE ROOM Mongo ID", "ROOM HASH ID",
		"groupingKey", "Status", "Notes",
	}
	if err := csvWriter.Write(header); err != nil {
		log.Fatalf("Failed to write CSV header: %v", err)
	}

	// Write records
	for _, result := range results {
		row := []string{
			result.PKFAREHotelID,
			result.PKFARERoomID,
			result.PKFARERoomName,
			result.ExpediaHotelID,
			result.ExpediaRoomID,
			result.ExpediaRoomName,
			result.PKFAREHotelHashID,
			result.PKFARERoomMongoID,
			result.RoomHashID,
			result.GroupingKey,
			result.Status,
			result.Notes,
		}
		if err := csvWriter.Write(row); err != nil {
			log.Fatalf("Failed to write CSV row: %v", err)
		}
	}

	log.Printf("Process completed. Processed %d records", len(results))
	log.Println("Output saved to: resource/PKFARE_Mapping_with_grouping_key_output.csv")

	// Step 5: Export SUCCESS records to a separate CSV with simplified format
	successOutputFile, err := os.Create("resource/PKFARE_Mapping_success_only.csv")
	if err != nil {
		log.Fatalf("Failed to create success-only CSV file: %v", err)
	}
	defer successOutputFile.Close()

	successCsvWriter := csv.NewWriter(successOutputFile)
	defer successCsvWriter.Flush()

	// Write header for success-only CSV
	successHeader := []string{"hotel_id", "room_id", "room_group_key"}
	if err := successCsvWriter.Write(successHeader); err != nil {
		log.Fatalf("Failed to write success CSV header: %v", err)
	}

	// Write only SUCCESS records to the new CSV
	for _, result := range results {
		if result.Status == "SUCCESS" {
			roomFilter := bson.M{
				"vendor.name":                  "PKFARE",
				"vendor.vendorHotelExternalId": result.PKFAREHotelID,
				"vendor.vendorExternalId":      result.PKFARERoomID,
			}

			projection := bson.M{
				"isDeleted": 1,
				"isActive":  1,
				"hotelId":   1,
				"vendor":    1,
			}

			var room Room
			roomCollection.FindOne(context.TODO(), roomFilter, &options.FindOneOptions{Projection: projection}).Decode(&room)
			successRow := []string{
				result.HashID,
				room.ID,
				result.GroupingKey,
			}
			if err := successCsvWriter.Write(successRow); err != nil {
				log.Fatalf("Failed to write success CSV row: %v", err)
			}
		}
	}

	// Print summary statistics
	successCount := 0
	failedCount := 0
	for _, result := range results {
		if result.Status == "SUCCESS" {
			successCount++
		} else {
			failedCount++
		}
	}

	log.Printf("Summary: %d successful, %d failed", successCount, failedCount)
	log.Println("Success-only output saved to: resource/PKFARE_Mapping_success_only.csv")
}
