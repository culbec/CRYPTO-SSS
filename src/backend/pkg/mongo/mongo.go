package mongo

import (
	"context"
	"errors"
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	config "github.com/culbec/CRYPTO-sss/src/backend/pkg"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClientInterface: interface specifying the methods for the client
type ClientInterface interface {
	QueryCollection(ctx context.Context, collectionName string, conditions *bson.D, opts *options.FindOptions, results any) (int, error)
	InsertDocument(ctx context.Context, collectionName string, conditions *bson.D, document any) (any, int, error)
	DeleteDocument(ctx context.Context, collectionName string, conditions *bson.D) (int, error)
	EditDocument(ctx context.Context, collectionName string, conditions *bson.D, document any) (int, error)
}

// ClientConfig: struct to hold the client configuration
type ClientConfig struct {
	DbURI  string
	DbName string
}

// Client: struct to hold the client connection and environment variables
type Client struct {
	ctx      context.Context
	dbClient *mongo.Client
	config   *ClientConfig
}

// QueryCollection: queries a named collection in the database based on some conditions.
// Returns an HTTP status code and an error.
func (client *Client) QueryCollection(collectionName string, conditions *bson.D, opts *options.FindOptions, results any) (int, error) {
	logger := logging.FromContext(client.ctx)

	// Accessing the collection
	collection := client.dbClient.Database(client.config.DbName).Collection(collectionName)
	logger.Info("Accessed collection", "collection", collection.Name())

	if conditions == nil {
		conditions = &bson.D{}
	}

	// Querying the collection
	cursor, err := collection.Find(client.ctx, conditions, opts)

	if err != nil {
		logger.Error("Error querying the collection", "error", err.Error())
		return http.StatusInternalServerError, err
	}

	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			logger.Error("Error closing the cursor", "error", err.Error())
		}
	}(cursor, client.ctx)

	// Decoding directly into a generic result
	if err = cursor.All(client.ctx, results); err != nil {
		logger.Error("Error decoding the results", "error", err.Error())
		return http.StatusInternalServerError, err
	}

	// Checking if the cursor encountered any errors
	if err = cursor.Err(); err != nil {
		logger.Error("Error with the cursor", "error", err.Error())
		return http.StatusInternalServerError, err
	}

	logger.Info("Query on collection OK", "collection", collection.Name())
	return http.StatusOK, nil
}

// InsertDocument: inserts a new document into the named collection. The conditions parameter is used to check for uniqueness.
// Returns the ID of the inserted document, HTTP status code, and an error
func (client *Client) InsertDocument(collectionName string, conditions *bson.D, document any) (any, int, error) {
	logger := logging.FromContext(client.ctx)

	collection := client.dbClient.Database(client.config.DbName).Collection(collectionName)
	logger.Info("Accessed collection", "collection", collection.Name())

	// Checking the insertion conditions
	// Mainly checking for uniqueness of the document
	if conditions != nil {
		cursor, err := collection.Find(client.ctx, conditions)

		if err != nil {
			logger.Error("Error querying the collection", "error", err.Error())
			return nil, http.StatusInternalServerError, err
		}

		defer func(cursor *mongo.Cursor, ctx context.Context) {
			err := cursor.Close(ctx)
			if err != nil {
				logger.Error("Error closing the cursor", "error", err.Error())
			}
		}(cursor, client.ctx)

		if cursor.Next(client.ctx) {
			logger.Info("Document already exists in the collection")
			return nil, http.StatusConflict, errors.New("document already exists in the collection")
		}
	}

	insertResult, err := collection.InsertOne(client.ctx, document)

	if err != nil {
		logger.Error("Error inserting the document", "error", err.Error())
		return nil, http.StatusInternalServerError, err
	}

	logger.Info("Inserted document", "id", insertResult.InsertedID)
	return insertResult.InsertedID.(primitive.ObjectID), http.StatusCreated, nil
}

// DeleteDocument: deletes a document from the named collection based on the conditions provided.
// Returns the HTTP status code and an error.
func (client *Client) DeleteDocument(collectionName string, conditions *bson.D) (int, error) {
	logger := logging.FromContext(client.ctx)

	collection := client.dbClient.Database(client.config.DbName).Collection(collectionName)
	logger.Info("Accessed collection", "collection", collection.Name())

	deletedResult, err := collection.DeleteOne(client.ctx, conditions)

	if err != nil {
		logger.Error("Error deleting the document", "error", err.Error())
		return http.StatusInternalServerError, err
	}

	// Checking if the document was actually deleted
	if deletedResult.DeletedCount == 0 {
		logger.Info("Document not found in the collection")
		return http.StatusBadRequest, errors.New("item not found, the ID might be incorrect")
	}

	return http.StatusOK, nil
}

// EditDocument: replaces a document in the named collection based on the conditions provided.
// Returns the HTTP status code and an error.
func (client *Client) EditDocument(collectionName string, conditions *bson.D, document any) (int, error) {
	logger := logging.FromContext(client.ctx)

	collection := client.dbClient.Database(client.config.DbName).Collection(collectionName)
	logger.Info("Accessed collection", "collection", collection.Name())

	replaceResult, err := collection.ReplaceOne(client.ctx, conditions, document)

	if err != nil {
		logger.Error("Error updating the document", "error", err.Error())
		return http.StatusInternalServerError, err
	}

	// Verifying if the document was actually updated
	if replaceResult.ModifiedCount == 0 {
		logger.Info("Document not found in the collection")
		return http.StatusBadRequest, errors.New("item not found, the ID might be incorrect or the item is the same")
	}

	return http.StatusOK, nil
}

// PrepareClient: prepares the client connection.
// Returns the client and an error if the connection fails.
func PrepareClient(ctx context.Context, config *config.Config) (*Client, error) {
	logger := logging.FromContext(ctx)

	if config.DbURI == "" {
		logger.Error("DB URI not set, skipping DB connection")
		return nil, errors.New("DB URI not set")
	}

	if config.DbName == "" {
		logger.Error("DB Name not set, skipping DB connection")
		return nil, errors.New("DB Name not set")
	}

	// Set the server API to stable version 1
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)

	// Initialize the client apiOptions
	apiOptions := options.Client().ApplyURI(config.DbURI).SetServerAPIOptions(serverAPIOptions)

	// Creating a new client and connecting to the server
	client, err := mongo.Connect(ctx, apiOptions)

	if err != nil {
		logger.Error("Error connecting to the server", "error", err.Error())
		return nil, err
	}

	// Ping the server to ensure the connection is established
	err = client.Ping(ctx, nil)
	if err != nil {
		logger.Error("Error pinging the server", "error", err.Error())
		return nil, err
	}

	return &Client{
		ctx:      ctx,
		dbClient: client,
		config: &ClientConfig{
			DbURI:  config.DbURI,
			DbName: config.DbName,
		},
	}, nil
}

// Cleanup: cleans up the client connection.
// Returns an error if the connection fails.
func Cleanup(client *Client) {
	logger := logging.FromContext(client.ctx)

	if err := client.dbClient.Disconnect(client.ctx); err != nil {
		logger.Error("Error disconnecting from the server", "error", err.Error())
		// Note: Consider returning error instead of panicking for better error handling
		panic(err)
	}
}
