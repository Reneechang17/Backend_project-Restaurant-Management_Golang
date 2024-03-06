package main

import (
	"os"

	"golang-Restaurant-Management-backend/database"
	middleware "golang-Restaurant-Management-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	routes "golang-Restaurant-Management-backend/routes"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())
	routes.UserRoutes(router)
	router.Use(middleware.Authentication())

	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	router.Run(":" + port)
}
