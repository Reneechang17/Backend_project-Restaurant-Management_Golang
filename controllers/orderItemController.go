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

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "OrderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Find all order items.
		result, err := orderItemCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing ordered items"})
		}

		// Decode the query results into a slice of bson.M objects.
		var allOrderItems []bson.M
		if err = result.All(ctx, &allOrderItems); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		orderItemId := c.Param("order_item_id")
		var orderItem models.OrderItem

	  // Find the order item in the database using the provided order_item_id.
		err := orderItemCollection.FindOne(ctx, bson.M{"orderItem_id": orderItemId}).Decode(&orderItem)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing ordered item"})
			return
		}

		c.JSON(http.StatusOK, orderItem)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")

		// Call the ItemsByOrder function to get all order items for the specified order.
		allOrderItems, err := ItemsByOrder(orderId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing order items by order ID"})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

// ItemsByOrder is a utility function that fetches order items based on the order ID.
// It performs an aggregation pipeline operation in MongoDB to fetch and format the required data.
func ItemsByOrder(id string) (OrderItems []primitive.M, err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

	// match a particular record with a particular key from database
	matchStage := bson.D{{"$match", bson.D{{"order_id", id}}}}
	// $lookup : is a function to look up from a particular collection
	// {"from", "food"} : where we look up from (from food collection)
	// {"localField", "food_id"} {"foreignField", "food_id"}: what's in my localField(OrderItem model) and foreignField(Food model)
	// as : means how do you want it to be represented as
	lookupStage := bson.D{{"$lookup", bson.D{{"from", "food"}, {"localField", "food_id"}, {"foreignField", "food_id"}, {"as", "food"}}}}
	// $unwind: takes a particular array, use and access it. Once we unwind it, mongoDb can perform some operations on it.
	// {"preserveNullAndEmptyArrays", true}: default is false
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$food"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupOrderStage := bson.D{{"$lookup", bson.D{{"from", "order"}, {"localField", "order_id"}, {"foreignField", "order_id"}, {"as", "order"}}}}
	unwindOrderStage := bson.D{{"$unwind", bson.D{{"path", "$order"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupTableStage := bson.D{{"$lookup", bson.D{{"from", "table"}, {"localField", "order.table_id"}, {"foreignField", "table_id"}, {"as", "table"}}}}
	unwindTableStage := bson.D{{"$unwind", bson.D{{"path", "$table"}, {"preserveNullAndEmptyArrays", true}}}}

	// projectStage: to manage the fields that you'll be turning to the frontend, means controls what goes to the next stage
	// because after we process the above(mathch, lookup, unwind), we will get lots of fields and data that not required, they might confuse the frontend
	projectStage := bson.D{
		{"$project", bson.D{
			{"id", 0},                 // 0 means do not goes to next stage
			{"amount", "$food.price"}, // which send to frontend and refer to price in Food model
			{"total_count", 1},        // 1 means should go to the frontend
			{"food_name", "$food.name"},
			{"food_image", "$food.food_image"},
			{"table_number", "$table.table_number"},
			{"table_id", "$table.table_id"},
			{"order_id", "$order.order_id"},
			{"price", "$food.price"},
			{"quantity", 1},
		}}}

	// groupStage : group all the data based on particular parameters
	groupStage := bson.D{{"$group", bson.D{{"_id", bson.D{{"order_id", "$order_id"}, {"table_id", "$table_id"}, {"table_number", "$table_number"}}}, {"payment_due", bson.D{{"$sum", "$amount"}}}, {"total_count", bson.D{{"$sum", 1}}}, {"order_items", bson.D{{"$push", "$$ROOT"}}}}}}

	projectStage2 := bson.D{
		{"$project", bson.D{

			{"id", 0},
			{"payment_due", 1},
			{"total_count", 1},
			{"table_number", "$_id.table_number"},
			{"order_items", 1},
		}}}

	// Execute an aggregation pipeline to fetch and format order items.
	result, err := orderItemCollection.Aggregate(ctx, mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2})

	if err != nil {
		panic(err)
	}

  // Decode the results from the aggregation pipeline.
  // The results are decoded into a slice of primitive.M, 
  // where each element of the slice is a map representing a BSON document (an order item in this case).
	if err = result.All(ctx, &OrderItems); err != nil {
		panic(err)
	}

	defer cancel()

	return OrderItems, err
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var orderItemPack OrderItemPack
		var order models.Order

		// Bind the JSON request body to the OrderItemPack struct.
		if err := c.BindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set the order date to the current time.
		order.Order_Date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		// Initialize a slice to hold the order items for batch insertion.（批量插入）
		orderItemToBeInserted := []interface{}{}
		// Assign the table ID from the order item pack to the order.
		order.Table_id = orderItemPack.Table_id
		// Create a new order and get its ID.
		order_id := OrderItemOrderCreator(order)

		// Iterate over the order items to process each one.
		for _, orderItem := range orderItemPack.Order_items {
			// Assign the generated order ID to each order item
			orderItem.Order_id = order_id

			// Validate the structure of each order item.
			validationErr := validate.Struct(orderItem)
			if validationErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
				return
			}

			// Generate a unique ID for each order item and set the created and updated timestamps.
			orderItem.ID = primitive.NewObjectID()
			orderItem.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Order_item_id = orderItem.ID.Hex()

			// Round the unit price to two decimal places.
			var num = toFixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &num
			// Add the order item to the slice for batch insertion.
			orderItemToBeInserted = append(orderItemToBeInserted, orderItem)
		}

		// Insert all the order items into the database at once.
		insertedOrderItems, err := orderItemCollection.InsertMany(ctx, orderItemToBeInserted)

		if err != nil {
			log.Fatal(err)
		}

		defer cancel()

		c.JSON(http.StatusOK, insertedOrderItems)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var orderItem models.OrderItem
		orderItemId := c.Param("order_item_id")

		var updateObj primitive.D

		// Append the update field
		if orderItem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{"unit_price", *&orderItem.Unit_price})
		}
		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{"quantity", *&orderItem.Quantity})
		}
		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"food_id", *&orderItem.Food_id})
		}

		// Update the 'updated_at' timestamp
		orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.Updated_at})

		// Set the upsert option to true, to create a new document if no document matches the filter.
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Define a filter to find the order item in the database.
		filter := bson.M{"order_item_id": orderItemId}

		// Perform the update operation on the database.
		result, err := orderItemCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"&set", updateObj},
			},
			&opt,
		)

		if err != nil {
			msg := fmt.Sprintf("Order item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()
		
		c.JSON(http.StatusOK, result)
	}
}
