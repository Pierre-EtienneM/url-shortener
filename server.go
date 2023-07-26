package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"encoding/json"
	"github.com/teris-io/shortid"
	"context"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// URL represents the URL document in the database
type URL struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	OriginalURL  string             `bson:"original_url"`
	ShortURL     string             `bson:"short_url"`
}

var collection *mongo.Collection

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Database Connection
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("mongodb+srv://pemorin:Secret12345@cluster0.hbflo.mongodb.net/?retryWrites=true&w=majority").SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)

	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(nil)

	err = client.Ping(nil, readpref.Primary())
	if err != nil {
		log.Fatal("Could not connect to the database:", err)
	}

	fmt.Println("Connected to MongoDB!")

	collection = client.Database("your-database-name").Collection("your-collection-name")

	r := mux.NewRouter()

	r.HandleFunc("/api/shorturl/new", createShortURL).Methods("POST")
	r.HandleFunc("/api/shorturl/{shortURL}", redirectToOriginalURL).Methods("GET")
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "views/index.html")
	})

	handler := cors.Default().Handler(r)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func createShortURL(w http.ResponseWriter, r *http.Request) {
	// Parse the JSON request body
	type RequestBody struct {
		URLInput string `json:"url_input"`
	}
	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the URL
	if !isValidURL(requestBody.URLInput) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Check if the URL is already in the database
	var result URL
	err = collection.FindOne(nil, bson.M{"original_url": requestBody.URLInput}).Decode(&result)
	if err == nil {
		// URL already exists in the database, return the existing short URL
		json.NewEncoder(w).Encode(map[string]string{
			"original_url": result.OriginalURL,
			"short_url":    result.ShortURL,
		})
		return
	} else if err != mongo.ErrNoDocuments {
		// Some other error occurred while querying the database
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Generate a new short URL code
	shortURLCode := shortid.MustGenerate()

	// Insert the URL into the database
	url := URL{
		OriginalURL: requestBody.URLInput,
		ShortURL:    shortURLCode,
	}
	_, err = collection.InsertOne(nil, url)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Return the created short URL
	json.NewEncoder(w).Encode(map[string]string{
		"original_url": requestBody.URLInput,
		"short_url":    shortURLCode,
	})
}

func redirectToOriginalURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortURLCode := vars["shortURL"]

	var result URL
	err := collection.FindOne(nil, bson.M{"short_url": shortURLCode}).Decode(&result)
	if err == nil {
		// Redirect to the original URL
		http.Redirect(w, r, result.OriginalURL, http.StatusSeeOther)
		return
	} else if err != mongo.ErrNoDocuments {
		// Some other error occurred while querying the database
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Short URL not found
	http.Error(w, "Short URL not found", http.StatusNotFound)
}

func isValidURL(url string) bool {
	// Implement your URL validation logic here
	// You can use the "valid-url" library or write your own validation
	// For example, using a regular expression
	// (Please note that this regular expression is a simple example and may not cover all cases)
	// urlRegex := regexp.MustCompile(`^[a-zA-Z]+://[^/]+`)
	// return urlRegex.MatchString(url)
	return true
}
