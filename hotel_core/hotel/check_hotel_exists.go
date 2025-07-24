package hotel

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Vendor struct {
	Name             string `bson:"name"`
	VendorExternalId string `bson:"vendorExternalId"`
	IsActive         int    `bson:"isActive"`
	RawIsDeleted     int    `bson:"rawIsDeleted"`
}

type Hotel struct {
	ID       string   `bson:"_id"`
	HotelId  string   `bson:"hotelId"`
	PublicId string   `bson:"publicId"`
	Vendors  []Vendor `bson:"vendors"`
}

type HotelRaw struct {
	ID      string `bson:"_id"`
	HotelId string `bson:"hotelId"`
	Vendor  string `bson:"vendor"`
}

type RoomRaw struct {
	ID          string `bson:"_id"`
	HotelId     string `bson:"hotelId"`
	Vendor      string `bson:"vendor"`
	RoomId      string `bson:"roomId"`
	IsPublished bool   `bson:"isPublished"`
	IsSynced    int    `bson:"isSynced"`
}

type Job struct {
	Record []string
}

const workerCount = 20

func CheckHotelExists() {
	clientOptions := options.Client().ApplyURI("mongodb://mfaji:Vxu4pesbH5@10.145.128.146:27017/admin")
	ctx := context.TODO()
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.TODO())

	database := client.Database("hotel_core")
	collection := database.Collection("hotel")
	collectionHtlRaw := database.Collection("hotel_raw")
	collectionRoomRaw := database.Collection("room_raw")

	inputFile, err := os.Open("/Users/gabrielevangeli/Documents/belajargolang/cek-case-hotel-exist/BCOM_AGODA_V2_17_July.csv")
	if err != nil {
		log.Fatal("Error opening CSV:", err)
	}
	defer inputFile.Close()

	reader := csv.NewReader(inputFile)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal("Error reading CSV:", err)
	}

	if len(records) == 0 || records[0][0] != "AGODA ID" || records[0][1] != "TARGET NEW VENDOR" {
		log.Fatal("Invalid CSV header")
	}

	outputFile, err := os.Create("/Users/melvinismanto/tiket/tiket-go/TIX-HOTEL-INVENTORY-SCRIPT/resource/bulk_merge_vendor_1/result-BCOM_AGODA_V2_17_July.csv")
	if err != nil {
		log.Fatal("Error creating output file:", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()
	writer.Write([]string{"old vendor", "old hotelVendorId", "old hotelHashId", "vendor", "hotelVendorId", "hotelHashId", "publicId", "remark", "room raw status to room"})

	jobChan := make(chan Job, len(records)-1)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				rows := processRecord(ctx, collection, collectionRoomRaw, collectionHtlRaw, job.Record)
				mu.Lock()
				for _, row := range rows {
					writer.Write(row)
				}
				writer.Flush()
				mu.Unlock()
			}
		}()
	}

	for _, record := range records[1:] {
		jobChan <- Job{Record: record}
	}
	close(jobChan)
	wg.Wait()

	fmt.Println("CSV successfully updated -> output.csv")
}

func isNotSyncInRoom(ctx context.Context, collectionRoomRaw *mongo.Collection, vendorName, hotelId string) string {
	var result RoomRaw
	filter := bson.M{
		"vendor":  vendorName,
		"hotelId": hotelId,
		"roomId": bson.M{
			"$exists": true,
		},
	}

	err := collectionRoomRaw.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return err.Error() + "room raw is not exist"
	}
	if !result.IsPublished && result.IsSynced == 0 {
		return "it is not picked yet"
	}
	if result.IsPublished && result.IsSynced == 1 {
		return "it is synced"
	}
	return "it is not picked yet"
}

func processRecord(ctx context.Context, collection, collectionRoomRaw, collectionHtlRaw *mongo.Collection, record []string) [][]string {
	hotelId := record[0]
	vendor := record[1]
	resultRows := [][]string{}

	filter := bson.M{
		"vendors": bson.M{
			"$elemMatch": bson.M{
				"name":             "AGODA",
				"vendorExternalId": hotelId,
			},
		},
	}
	fmt.Println("check this hotel id", hotelId)

	var result Hotel
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		filter = bson.M{
			"vendors": bson.M{
				"$elemMatch": bson.M{
					"name":             vendor,
					"vendorExternalId": hotelId,
				},
			},
		}
		var resultHotel Hotel
		err := collection.FindOne(ctx, filter).Decode(&resultHotel)
		if err != nil {
			var resultRaw HotelRaw
			filterHtlRaw := bson.M{
				"hotelId": hotelId,
				"vendor":  vendor,
			}
			err := collectionHtlRaw.FindOne(ctx, filterHtlRaw).Decode(&resultRaw)
			if err != nil {
				resultRows = append(resultRows, []string{"AGODA", "no data exist in hotel", "no data exist in hotel", vendor, hotelId, "no data exist in hotel raw", "no data exist in hotel raw", "no data exist in hotel raw", ""})
			} else {
				resultRows = append(resultRows, []string{"AGODA", "no data exist in hotel", "no data exist in hotel", vendor, hotelId, "unmapped", "unmapped", "unmapped", ""})
			}
			return resultRows
		}

		found := false
		foundIdentics := false
		vendorIdentics := []string{}
		hotelIdIdentics := []string{}
		vendorIdentic := ""
		hotelIdIdentic := ""

		for _, v := range resultHotel.Vendors {
			if v.Name == "AGODA" {
				if hotelId == v.VendorExternalId {
					found = true
					foundIdentics = true
					vendorIdentic = v.Name
					hotelIdIdentic = v.VendorExternalId
				}
				continue
			}
			if v.Name == "AGODA" || v.Name == "AGODA_V2" || v.Name == "BCOM" {
				if hotelId != v.VendorExternalId {
					found = true
					vendorIdentics = append(vendorIdentics, v.Name)
					hotelIdIdentics = append(hotelIdIdentics, v.VendorExternalId)
				}
			}
		}

		if found {
			if foundIdentics && len(vendorIdentics) == 0 {
				resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentic, hotelIdIdentic)
				resultRows = append(resultRows, []string{"AGODA", hotelId, resultHotel.HotelId, vendorIdentic, hotelIdIdentic, resultHotel.HotelId, resultHotel.PublicId, "no duplicate", resultRoom})

			}
			if foundIdentics && len(vendorIdentics) > 0 {
				resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentic, hotelIdIdentic)
				resultRows = append(resultRows, []string{"AGODA", hotelId, resultHotel.HotelId, vendorIdentic, hotelIdIdentic, resultHotel.HotelId, resultHotel.PublicId, "possible merge same vendor", resultRoom})
			}
			for i := range vendorIdentics {
				resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentics[i], hotelIdIdentics[i])
				resultRows = append(resultRows, []string{vendorIdentics[i], hotelIdIdentics[i], resultHotel.HotelId, vendor, hotelId, resultHotel.HotelId, resultHotel.PublicId, "Hotels mapped to the same property_hash_id but with different AGODA vendor_hotel_id", resultRoom})
			}
		} else {
			resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendor, hotelId)
			resultRows = append(resultRows, []string{"AGODA", "no data exist in hotel", "no data exist in hotel", vendor, hotelId, resultHotel.HotelId, resultHotel.PublicId, vendor + " is a new member of this property_hash_id", resultRoom})
		}

		return resultRows
	}

	found := false
	foundIdentics := false
	vendorIdentics := []string{}
	hotelIdIdentics := []string{}
	vendorIdentic := ""
	hotelIdIdentic := ""

	for _, v := range result.Vendors {
		if v.Name == "AGODA_V2" || v.Name == "BCOM" {
			if hotelId == v.VendorExternalId {
				found = true
				foundIdentics = true
				vendorIdentic = v.Name
				hotelIdIdentic = v.VendorExternalId
				continue
			}
		}
		if v.Name == "AGODA" || v.Name == "AGODA_V2" || v.Name == "BCOM" {
			if hotelId != v.VendorExternalId {
				found = true
				vendorIdentics = append(vendorIdentics, v.Name)
				hotelIdIdentics = append(hotelIdIdentics, v.VendorExternalId)
			}
		}
	}

	if !found {
		filter := bson.M{
			"vendors": bson.M{
				"$elemMatch": bson.M{
					"name":             vendor,
					"vendorExternalId": hotelId,
				},
			},
		}
		var resultHotel Hotel
		err := collection.FindOne(ctx, filter).Decode(&resultHotel)
		if err != nil {
			var resultRaw HotelRaw
			filterHtlRaw := bson.M{
				"hotelId": hotelId,
				"vendor":  vendor,
			}
			err := collectionHtlRaw.FindOne(ctx, filterHtlRaw).Decode(&resultRaw)
			if err != nil {
				resultRows = append(resultRows, []string{"AGODA", hotelId, result.HotelId, vendor, hotelId, "no data exist in hotel raw", "no data exist in hotel raw", "no data exist in hotel raw"})
			} else {
				resultRows = append(resultRows, []string{"AGODA", hotelId, result.HotelId, vendor, hotelId, "unmapped", "unmapped", "unmapped"})
			}
			return resultRows
		}
		resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendor, hotelId)
		resultRows = append(resultRows, []string{"AGODA", hotelId, result.HotelId, vendor, hotelId, resultHotel.HotelId, resultHotel.PublicId, "possible duplicate property", resultRoom})
		return resultRows
	}

	if foundIdentics && len(vendorIdentics) == 0 {
		resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentic, hotelIdIdentic)
		resultRows = append(resultRows, []string{"AGODA", hotelId, result.HotelId, vendorIdentic, hotelIdIdentic, result.HotelId, result.PublicId, "no duplicate", resultRoom})
	}
	if foundIdentics && len(vendorIdentics) > 0 {
		resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentic, hotelIdIdentic)
		resultRows = append(resultRows, []string{"AGODA", hotelId, result.HotelId, vendorIdentic, hotelIdIdentic, result.HotelId, result.PublicId, "possible merge same vendor", resultRoom})
	}
	for i := range vendorIdentics {
		resultRoom := isNotSyncInRoom(ctx, collectionRoomRaw, vendorIdentics[i], hotelIdIdentics[i])
		resultRows = append(resultRows, []string{vendorIdentics[i], hotelIdIdentics[i], result.HotelId, vendor, hotelId, result.HotelId, result.PublicId, "Hotels mapped to the same property_hash_id but with different AGODA vendor_hotel_id", resultRoom})
	}

	return resultRows
}
