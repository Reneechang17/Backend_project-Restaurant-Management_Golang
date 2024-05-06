// Use Gin web framework in Go and MongoDB as database.
package main

import (
	"os"

	database "golang-Restaurant-Management-backend/database"
	middleware "golang-Restaurant-Management-backend/middleware"
	routes "golang-Restaurant-Management-backend/routes"

	"github.com/gin-gonic/gin" // import Gin Web Framework
	"go.mongodb.org/mongo-driver/mongo" // use mongo-driver to connect
)

// set foodCollection as a reference to the 'food' collection in MongoDB
// foodCollection as a global variable holds the reference point to the 'food' collection in MongoDB
var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

func main() {
	port := os.Getenv("PORT")

	// default port is 8000
	if port == "" {
		port = "8000"  
	}

	// create a new Gin router and use the built-in logging middleware on Gin
	router := gin.New()
	router.Use(gin.Logger())

	// set up routes related to user and use our middleware.
	routes.UserRoutes(router)
	router.Use(middleware.Authentication())

	// set up another routes
	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	// start the gin server and listen on the 8000 port
	router.Run(":" + port)
}
