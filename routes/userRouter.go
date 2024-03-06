package routes

import (
	"github.com/gin-gonic/gin"
	controller "golang-Restaurant-Management-backend/controllers"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	// calls the GetUsers function by controller package when the server receives a GET request at URL
	incomingRoutes.GET("/users", controller.GetUsers())
	incomingRoutes.GET("/users/:user_id", controller.GetUser())
	incomingRoutes.POST("/users/signup", controller.SignUp())
	incomingRoutes.POST("/users/login", controller.Login())
}
