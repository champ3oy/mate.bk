package config

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var Database *mongo.Database

// Connect to MongoDB
func ConnectToDB() {
	clientOptions := options.Client().ApplyURI(MongoURI)
	var err error
	Client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	Database = Client.Database(MongoDBName)
	fmt.Println("Successfully connected to MongoDB")
}

// InsertOne - Insert a single document into the collection
func InsertOne(collectionName string, document interface{}) error {
	collection := Database.Collection(collectionName)
	_, err := collection.InsertOne(context.Background(), document)
	return err
}

// FindOne - Find a single document in the collection
func FindOne(collectionName string, filter bson.M) (*mongo.SingleResult, error) {
	collection := Database.Collection(collectionName)
	result := collection.FindOne(context.Background(), filter)
	return result, result.Err()
}

// Find - Find multiple documents in the collection
func Find(collectionName string, filter bson.M, options *options.FindOptions) (*mongo.Cursor, error) {
	collection := Database.Collection(collectionName)
	cursor, err := collection.Find(context.Background(), filter, options)
	return cursor, err
}

// UpdateOne - Update a single document in the collection
func UpdateOne(collectionName string, filter bson.M, update bson.M) (*mongo.UpdateResult, error) {
	collection := Database.Collection(collectionName)
	result, err := collection.UpdateOne(context.Background(), filter, update)
	return result, err
}

// UpdateMany - Update multiple documents in the collection
func UpdateMany(collectionName string, filter bson.M, update bson.M) (*mongo.UpdateResult, error) {
	collection := Database.Collection(collectionName)
	result, err := collection.UpdateMany(context.Background(), filter, update)
	return result, err
}

// DeleteOne - Delete a single document in the collection
func DeleteOne(collectionName string, filter bson.M) (*mongo.DeleteResult, error) {
	collection := Database.Collection(collectionName)
	result, err := collection.DeleteOne(context.Background(), filter)
	return result, err
}

// DeleteMany - Delete multiple documents in the collection
func DeleteMany(collectionName string, filter bson.M) (*mongo.DeleteResult, error) {
	collection := Database.Collection(collectionName)
	result, err := collection.DeleteMany(context.Background(), filter)
	return result, err
}

// CountDocuments - Count the number of documents matching the filter
func CountDocuments(collectionName string, filter bson.M) (int64, error) {
	collection := Database.Collection(collectionName)
	count, err := collection.CountDocuments(context.Background(), filter)
	return count, err
}

// Aggregate - Perform aggregation operations
func Aggregate(collectionName string, pipeline interface{}, options *options.AggregateOptions) (*mongo.Cursor, error) {
	collection := Database.Collection(collectionName)
	cursor, err := collection.Aggregate(context.Background(), pipeline, options)
	return cursor, err
}
