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

var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")

func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Query the menuCollection to find all documents.
		result, err := menuCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing the menu items."})
		}

		// Create a slice to hold the data for all menus.
		var allMenus []bson.M
		if err = result.All(ctx, &allMenus); err != nil {
			log.Fatal(err)
		}

		// Send the list of all menus as a JSON format.
		c.JSON(http.StatusOK, allMenus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		menuId := c.Param("menu_id")
		var menu models.Menu

		// Find the menu with the menu_id in the database and decode the result.
		err := foodCollection.FindOne(ctx, bson.M{"menu_id": menuId}).Decode(&menu)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while fetching the menu"})
		}

		c.JSON(http.StatusOK, menu)
	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var menu models.Menu

		// Bind the incoming JSON data to the menu struct.
		if err := c.BindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the menu struct
		validationErr := validate.Struct(menu)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Set the creation and update timestamps.
		menu.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		// Generate a new ID.
		menu.ID = primitive.NewObjectID()
		menu.Menu_id = menu.ID.Hex()

		// Insert new one.
		result, insertErr := menuCollection.InsertOne(ctx, menu)
		if insertErr != nil {
			msg := fmt.Sprintf("Menu item was not created.")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

// checks if the 'check' time is between 'start' and 'end' times.
func inTimeSpan(start, end, check time.Time) bool {
	return start.After(time.Now()) && end.After(start)
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var menu models.Menu

		// get the menu_id from the URL parameters and bind the incoming JSON data to the menu struct		
		if err := c.BindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		menuId := c.Param("menu_id")


		var updateObj primitive.D

		if menu.Start_Date != nil && menu.End_Date != nil {
			// first check id the time between start and end
			// If it's not, return an error message.
			if !inTimeSpan(*menu.Start_Date, *menu.End_Date, time.Now()) {
				msg := "Kindly retype the time."
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				defer cancel()
				return
			}

			// Add the start and end dates to the update object.
			updateObj = append(updateObj, bson.E{"start_date", menu.Start_Date})
			updateObj = append(updateObj, bson.E{"end_date", menu.End_Date})

      // Append updated fields to the update object.
			if menu.Name != "" {
				updateObj = append(updateObj, bson.E{"name", menu.Name})
			}
			if menu.Category != "" {
				updateObj = append(updateObj, bson.E{"name", menu.Category})
			}

			// Update the updated_at timestamp.
			menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			updateObj = append(updateObj, bson.E{"updated_at", menu.Updated_at})

			// try to update the table record in database(upsert/opt/filter)
		  // Set the upsert option to true, to create a new document if no document matches the filter.
			upsert := true
			opt := options.UpdateOptions{
				Upsert: &upsert,
			}

			filter := bson.M{"menu_id": menuId}

			result, err := menuCollection.UpdateOne(
				ctx,
				filter,
				bson.D{
					{"&set", updateObj},
				},
				&opt,
			)
			
			if err != nil {
				msg := "Menu update failed"
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			}

			defer cancel()
			
			c.JSON(http.StatusOK, result)
		}
	}
}
