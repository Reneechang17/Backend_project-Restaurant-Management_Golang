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

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Query the invoiceCollection to find all documents.
		result, err := invoiceCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing order items"})
		}
		// Create a slice to hold the data for all invoices.
		var allInvoices []bson.M
		if err = result.All(ctx, &allInvoices); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		invoiceId := c.Param("invoice_id")
		var invoice models.Invoice

		// Find the invoice with the invoice_id in the database and decode the result.
		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occur while listing invoice item"})
		}

		// use to hold the invoice view details.
		var invoiceView InvoiceViewFormat

		// Retrieve all order items associated with the invoice's order ID.
		allOrderItems, err := ItemsByOrder(invoice.Order_id)

		// Assign order ID and payment due date from the invoice to the invoice view.
		invoiceView.Order_id = invoice.Order_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		// Set default payment method to "null" and update if available.
		invoiceView.Payment_method = "null"
		if invoice.Payment_method != nil {
			invoiceView.Payment_method = *invoice.Payment_method
		}

		// Populate the remaining fields of the invoice view with the invoice data
		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = *&invoice.Payment_status

		// Extract specific details from the first order item for the invoice view.
		invoiceView.Payment_due = allOrderItems[0]["Payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		// Return the invoice view as a JSON response.
		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var invoice models.Invoice

		// Bind the incoming JSON data to the struct.
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create an order struct to hold order data.
		var order models.Order

		// Find the order in the database using the order ID from the invoice.
		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("message:Order was not found")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// Default the payment status
		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		// Set the payment due date, creation date, and update date
		invoice.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))
		invoice.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		// Generate and set a new ID.
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		// Validate the invoice struct. （make sure this is the OnlyOne ID)
		validationErr := validate.Struct(invoice)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Insert the new invoice into the database.
		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			msg := fmt.Sprintf("Invoice item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var invoice models.Invoice
		invoiceId := c.Param("invoice_id")

		// Bind the incoming JSON data to the invoice struct.
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		// Append the update fields.
		if invoice.Payment_method != nil {
			updateObj = append(updateObj, bson.E{"payment_method", invoice.Payment_method})
		}
		if invoice.Payment_status != nil {
			updateObj = append(updateObj, bson.E{"Payment_status", invoice.Payment_status})
		}

		// Update the 'updated_at' timestamp 
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", invoice.Updated_at})

		// Default the payment status 
		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		// try to update the table record in database(upsert/opt/filter)
    // Set the upsert option to true.
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Define the filter to find the invoice by invoice_id.
		filter := bson.M{"invoice_id": invoiceId}

		result, err := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)
		
		if err != nil {
			msg := fmt.Sprintf("Invoice item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}
