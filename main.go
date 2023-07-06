package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Define the LastTenDirectorsDealings struct
type LastTenDirectorsDealings struct {
	StockCode   string  `json:"stock_code"`
	Date        string  `json:"date"`
	Beneficiary string  `json:"Beneficiary"`
	DealType    string  `json:"deal_type"`
	Value       int64   `json:"value"`
	Volume      int64   `json:"volume"`
	Price       float32 `json:"price"`
}

func main() {

	// MongoDB Atlas connection settings
	mongoURI := "mongodb+srv://<user>:<password>@<collection>.mongodb.net/<database>retryWrites=true&w=majority"

	// Connect to MongoDB Atlas
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB Atlas: %v", err)
	}
	defer client.Disconnect(ctx)

	// Access the database and collection
	db := client.Database("<database>")
	collection := db.Collection("<collection>")

	// Instantiate default collector
	c := colly.NewCollector(
		// Visit only domains: moneyweb.co.za, www.moneyweb.co.za
		colly.AllowedDomains("moneyweb.co.za", "www.moneyweb.co.za"),
	)

	// Set up the callback function to handle the HTML response
	c.OnHTML(`div[id=cac-page]`, func(e *colly.HTMLElement) {
		// Extract the required data from the HTML elements
		e.ForEach(".sens-row.cac", func(_ int, el *colly.HTMLElement) {

			// Skip headings row
			if el.Index > 1 {
				dealings := LastTenDirectorsDealings{
					StockCode: "SSW",
				}

				// We know that DATE, DEAL TYPE, VALUE and VOLUME are all under
				// .col-lg-2.col-md-2 elements. Work through each element accordingly.
				el.ForEach(".col-lg-2.col-md-2", func(_ int, el_col *colly.HTMLElement) {
					switch el_col.Index {
					case 0:
						dealings.Date = el_col.Text
					case 1:
						dealings.DealType = el_col.Text
					case 2:
						value, err := strconv.ParseInt(strings.Replace(el_col.Text, ",", "", -1), 10, 0)
						if err != nil {
							log.Fatalf("Failed to convert element: %v", err)
						}
						dealings.Value = value
					case 3:
						volume, err := strconv.ParseInt(strings.Replace(el_col.Text, ",", "", -1), 10, 0)
						if err != nil {
							log.Fatalf("Failed to convert element: %v", err)
						}
						dealings.Volume = volume
					}
				})

				// We know that BENEFICIARY is the only one under a
				// .col-lg-3.col-md-3 element.
				dealings.Beneficiary = el.ChildText(".col-lg-3.col-md-3")

				// We know that PRICE is the only one under a
				// .col-lg-1.col-md-1.clear-padding element.
				element_text := el.ChildText(".col-lg-1.col-md-1.clear-padding")
				price, err := strconv.ParseFloat(strings.Replace(element_text, ",", "", -1), 32)
				if err != nil {
					log.Fatalf("Failed to convert element: %v", err)
				}
				dealings.Price = float32(price)

				// Convert the struct to JSON
				jsonData, err := json.Marshal(dealings)
				if err != nil {
					log.Fatalf("Failed to marshal JSON: %v", err)
				}

				// Insert the data into MongoDB
				_, err = collection.InsertOne(ctx, dealings)
				if err != nil {
					log.Fatalf("Failed to insert data into MongoDB: %v", err)
				}

				// Print the extracted data in JSON format
				fmt.Println(string(jsonData))
			}

		})

	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://moneyweb.co.za
	c.Visit("https://www.moneyweb.co.za/tools-and-data/click-a-company/SSW/")
}
