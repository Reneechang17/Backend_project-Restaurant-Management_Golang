package routes

// Routes used to define the URL routes and assign the req to the correspond controller

import(
	"github.com/gin-gonic/gin"

	// Use Controller to deal with specific HTTP request
	controller "golang-Restaurant-Management-backend/controllers"
)
// '*gin.Engine' used to represent the web application in this project
func FoodRoutes(incomingRoutes *gin.Engine){
	incomingRoutes.GET("/foods", controller.GetFoods()) // Get all food lists
	incomingRoutes.GET("/foods/:food_id", controller.GetFood()) // Get one food's info by specific ID
	incomingRoutes.POST("/foods", controller.CreateFood()) // Create a food item
	incomingRoutes.PATCH("/foods/:food_id", controller.UpdateFood()) // Update existed food item
}