package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// creates and returns a MongoDB client instance.
func DBinstance() *mongo.Client {
	MongoDb := "mongodb://localhost:27017"  // MongoDB connection string.
	fmt.Print(MongoDb)

	// Create a new MongoDB client.
	client, err := mongo.NewClient(options.Client().ApplyURI(MongoDb))
	if err != nil {
		log.Fatal(err)
	}
	// Create a context with a timeout for connecting to MongoDB.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect the client to MongoDB.
	err = client.Connect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected to mongodb")
	return client // Return the MongoDB client.
}

var Client *mongo.Client = DBinstance()

// returns a reference to a MongoDB collection.
func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	var collection *mongo.Collection = client.Database("restaurant").Collection(collectionName)

	return collection
}

