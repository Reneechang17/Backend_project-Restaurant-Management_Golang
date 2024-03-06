package controllers

import (
	"context"
	"fmt"
	"golang-Restaurant-Management-backend/database"
	"golang-Restaurant-Management-backend/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)
var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Query the orderCollection to find all documents.
		result, err := orderCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing order items"})
		}

		// Create a slice to hold the data for all orders.
		var allOrders []bson.M
		if err = result.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}

    // Send the list of all orders as a JSON response.
		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		orderId := c.Param("order_id")
		var order models.Order

		// Find the order with the table_id in the database and decode the result.
		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		defer cancel()
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error":"Error occur while fetching the orders"})
		}
		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var table models.Table
		var order models.Order

		// Bind the incoming JSON data to the order struct.
		if err := c.BindJSON(&order); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the order struct
		validationErr := validate.Struct(order)
		if validationErr != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}
		
		// Validate whether the specific tableId in order exist in database
		if order.Table_id != nil{
			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			defer cancel()
			if err != nil{
				msg := fmt.Sprintf("message:Table was not found")
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
		}

		// Set the creation and update timestamps 
		order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		// Generate a new unique ID
		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		// Insert the new one into the database.
		result, insertErr := orderCollection.InsertOne(ctx, order)
		if insertErr != nil{
			msg := fmt.Sprintf("Order item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error":msg})
			return
		}
		defer cancel()

		// Return the result of the insertion.
		c.JSON(http.StatusOK, result)

	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var table models.Table
		var order models.Order

		var updateObj primitive.D

		// get the order_id from the URL parameters and bind the incoming JSON data to the order struct
		orderId := c.Param("order_id")
		if err := c.BindJSON(&order); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// check if the tableID(menu) exist and if true, we can use the tableID when we update the tableID(menu)
		if order.Table_id != nil{
			err := menuCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			defer cancel()
			if err != nil{
				msg := fmt.Sprintf("message:Menu was not found")
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
			updateObj = append(updateObj, bson.E{"menu", order.Table_id})
		}

		// Update the updated_at timestamp.
		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", order.Updated_at})

	  // try to update the table record in database(upsert/opt/filter)
		// Set the upsert option to true, to create a new document if no document matches the filter.
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Define the filter to find the order by order_id.
		filter := bson.M{"order_id": orderId}

		// Update the order in the database.
		result, err := orderCollection.UpdateOne(
			ctx, 
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)

		if err != nil{
			msg := fmt.Sprintf("Order item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

// use to create a  new orderID and return it
func OrderItemOrderCreator(order models.Order) string{
	// Set the created and updated timestamps
	order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	// Generate a new unique ObjectID for the order and set it as the order's ID.
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	// Insert the new order into the database.
	orderCollection.InsertOne(ctx, order)
	defer cancel()

	// Return the newly created order ID.
	return order.Order_id
}
