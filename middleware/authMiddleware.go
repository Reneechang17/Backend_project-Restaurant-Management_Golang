package middleware

import (
	"fmt"
	helper "golang-Restaurant-Management-backend/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Authentication() gin.HandlerFunc {
	return func(c *gin.Context){
		// Retrieve the token from the request header.
		clientToken := c.Request.Header.Get("token")
		if clientToken == ""{
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("No Authentication Header provided")})
			c.Abort()
			return
		}

		// Validate the token and get the claims.
		claims, err := helper.ValidateToken(clientToken)
		if err != ""{
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			c.Abort()
			return
		}

		// Set user information in the context from the claims.
		c.Set("email", claims.Email)
		c.Set("first_name", claims.First_name)
		c.Set("last_name", claims.Last_name)
		c.Set("uid", claims.Uid)

		c.Next() // Proceed to the next handler in the chain.

	}
}
