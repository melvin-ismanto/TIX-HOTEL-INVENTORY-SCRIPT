package bulk_merge_vendor

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HotelResult struct {
	ID          primitive.ObjectID `bson:"_id"`
	UpdatedDate time.Time          `bson:"updatedDate"`
}

func FindSlowHotelMerge() {
	// MongoDB connection
	clientOpts := options.Client().ApplyURI("mongodb://mfaji:Vxu4pesbH5@10.145.128.146:27017/admin")
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect(context.TODO())

	collection := client.Database("hotel_core").Collection("bulk_merge_hotel_result")

	// Time range
	// startTime, _ := time.Parse(time.RFC3339Nano, "2025-07-23T06:40:32.319Z")
	// endTime, _ := time.Parse(time.RFC3339Nano, "2025-07-23T06:49:32.319Z")

	// Filter and sort
	firstId, _ := primitive.ObjectIDFromHex("68806833c3083074f8ebdabd")
	lastId, _ := primitive.ObjectIDFromHex("68807a7fc3083074f8911da7")

	filter := bson.M{
		"activityId": "6813e601-e37a-4da3-bd47-027119179a88",
		// "updatedDate": bson.M{
		// 	"$gte": startTime,
		// 	"$lt":  endTime,
		// },
		"_id": bson.M{
			"$gte": firstId,
			"$lte": lastId,
		},
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})

	cursor, err := collection.Find(context.TODO(), filter, findOpts)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer cursor.Close(context.TODO())

	var results []HotelResult
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatalf("Cursor decode error: %v", err)
	}

	log.Println("Compare updatedDate of each document to previous one")
	for i := 1; i < len(results); i++ {
		prev := results[i-1].UpdatedDate
		curr := results[i].UpdatedDate
		diff := curr.Sub(prev).Seconds()

		// Check if it processed more than 10 seconds
		if diff > 10 {
			fmt.Printf("ID: %s | updatedDate: %s | diff: %.2f seconds\n",
				results[i].ID.Hex(), curr.Format(time.RFC3339), diff)
		}
	}
	log.Println("## done ##")
}
