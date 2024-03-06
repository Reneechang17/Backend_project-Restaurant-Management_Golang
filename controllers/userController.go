package controllers

import (
	"context"
	"fmt"
	"golang-Restaurant-Management-backend/database"
	helper "golang-Restaurant-Management-backend/helpers"
	"golang-Restaurant-Management-backend/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// create a context with a timeout to avoid long-running queries(after 100 sec)
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Parse the recordPerPage and page query parameters, defaulting to 10 and 1 respectively
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}
		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil || page < 1 {
			page = 1
		}

		// Calculate the startIndex for MongoDB's pagination
		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		// Define the match and project stages for the MongoDB aggregation pipeline
		matchStage := bson.D{{"$match", bson.D{{}}}}
		projectStage := bson.D{
			{"$project", bson.D{
				{"_id", 0},
				{"total_count", 1},
				{"user_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordPerPage}}}},
			}}}

		// Perform the aggregation query
		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, projectStage})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing user items"})
		}

    // Decode the query results into a slice of bson.M objects.
		var allUsers []bson.M
		if err = result.All(ctx, &allUsers); err != nil {
			log.Fatal(err)
		}

		// Respond with the retrieved user data in JSON format.
		c.JSON(http.StatusOK, allUsers[0])
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		userId := c.Param("user_id")
		var user models.User

		// Find the user with the user_id in the database and decode the result into the user variable.
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		// Ensure the context is cancelled when the function exits.
		defer cancel()

		// handle error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing user items"})
		}

		c.JSON(http.StatusOK, user)
	}
}

// SignUp function process user sign-up, data validation, hash password, check email and phone num, generate tokens and insert user data into database
func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		// convert the JSON data coming from postman to something that golang understands
		// Bind the incoming JSON data to the user struct
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		// validate the data based on user struct
		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// check if the email has already been used by another user
		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		defer cancel()
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while checking for the email"})
			return
		}

		// hash pwd
		password := HashPassword(*user.Password)
		user.Password = &password

		// check if the phone num has already been used by another user
		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		defer cancel()
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while checking for the phone"})
			return
		}

		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "This email or phone number is already exists"})
			return
		}

		// create some extra details for the user object: created_at, updated_at, ID
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		// generate token and refresh token (generate all tokens function from Helper)
		token, refreshToken, _ := helper.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, user.User_id)
		user.Token = &token
		user.Refresh_Token = &refreshToken

		// if above are all ok, insert this user to user collection
		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil{
			msg := fmt.Sprintf("User item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()

		// return status OK and send the result back
		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User  // user data from client side.
		var foundUser models.User // find data from database

		// convert the login data from postman which is in JSON to golang readable format
		// Bind the incoming JSON data to the user struct
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		// find a user with that email and see if that user even exists
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
			return
		}

		// verify the password
		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		defer cancel()
		if passwordIsValid != true{
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// If the login is successful, generate new tokens.
		token, refreshToken, _ := helper.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, foundUser.User_id)

		// update tokens - token and refresh the token
		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		// return statusOK and user data
		c.JSON(http.StatusOK, foundUser)
	}
}

// use in SignUp
func HashPassword(password string) string {
	// Encrypt the password using bcrypt with a cost of 14.
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil{
		log.Panic(err) // If there is an error in password encryption, log it.
	}
	return string(bytes)
}

// use in Login to compare the provided password with the hashed password stored in the database.
func VerifyPassword(userPassword string, providePassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providePassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil{
		msg = fmt.Sprintf("Login or password is incorrect")
		check = false
	}
	return check, msg
}
