package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/google/generative-ai-go/genai"
	"github.com/labstack/echo/v4"
	"github.com/rs/cors"
	"google.golang.org/api/option"
)

var db *sql.DB

func connectDB() *sql.DB {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "root",
		Net:                  "tcp",
		Addr:                 "localhost:3306",
		DBName:               "chatatm",
		AllowNativePasswords: true,
	}
	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
	// Now you can execute SQL queries using the 'db' object
	return db
}
func uploadHandler(c echo.Context) error {
	print("Hello")
	// Read form data including file
	_, err := c.MultipartForm()
	if err != nil {
		return err
	}

	// Retrieve the file
	file, handler, err := c.Request().FormFile("image")
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a temporary file to save the uploaded file
	tempFile, err := os.CreateTemp("", "uploaded-image-*.jpg")
	if err != nil {
		return err
	}
	defer tempFile.Close()

	// Copy the file data to the temporary file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return err
	}

	// Print information about the uploaded file
	fmt.Printf("Received file: %s\n", handler.Filename)
	fmt.Printf("File size: %d bytes\n", handler.Size)

	// Optionally, you can process the file further here
	// For example, save it to a permanent location, process it, etc.

	// Use the path of the uploaded image to generate content
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("AIzaSyDQqEWt9KFtbEtSRYgipFkWO7t30nMKdKo"))
	if err != nil {
		return err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro-vision")
	imgData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return err
	}

	prompt := []genai.Part{
		genai.ImageData("jpeg", imgData),
		genai.Text("Give me the details of the flower such as color, size,scientific name,category,other_names,habitat,distribution,etymology,symbolism,uses,interesting facts etc. so that i can show the various categories to the user give the data in json format to show different categories of flowers.and values way give me more than 20 categories of specifications of the flowers in the image and please make the categories understandable by common man"),
	}
	res, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return err
	}

	printResponse(res)

	return c.String(http.StatusOK, "Image uploaded successfully and content generated.")
}

func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Println(part)
			}
		}
	}
	fmt.Println("---")
}

func main() {

	e := echo.New()
	db := connectDB()

	e.Use(echo.WrapMiddleware(cors.Default().Handler))


	e.GET("/", func(c echo.Context) error {
		ctx := context.Background()
		// Access your API key as an environment variable (see "Set up your API key" above)
		client, err := genai.NewClient(ctx, option.WithAPIKey("AIzaSyDQqEWt9KFtbEtSRYgipFkWO7t30nMKdKo"))

		if err != nil {
			log.Fatal(err)
		}
		defer client.Close()
		model := client.GenerativeModel("gemini-pro-vision")
		imgData1, _ := os.ReadFile("./cookie.jpg")
		prompt := []genai.Part{
			genai.ImageData("jpeg", imgData1),
			genai.Text("Describe about the image"),
		}
		res, err := model.GenerateContent(ctx, prompt...)

		if err != nil {
			log.Fatal(err)
		}
		printResponse(res)
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.POST("/forget-password", func(c echo.Context) error {
		// Parse the request body into a map
		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return c.String(http.StatusBadRequest, "Invalid request body")
		}
	
		// Check if the required fields are present in the request body
		if data["username"] == "" || data["email"] == "" {
			return c.String(http.StatusBadRequest, "Username and email are required")
		}
	
		// Query the database to check if the provided username and email match
		rows, err := db.Query("SELECT username FROM user WHERE username = ? AND email = ?", data["username"], data["email"])
		if err != nil {
			return c.String(http.StatusInternalServerError, "Database error")
		}
		defer rows.Close()
	
		// Check if the user exists with the provided username and email
		userExists := false
		for rows.Next() {
			userExists = true
			break
		}
	
		if !userExists {
			return c.String(http.StatusNotFound, "User not found")
		}
	
		// If everything is successful, return a success response
		return c.String(http.StatusOK, "Password recovery initiated")
	})
	

	e.POST("/send-data", func(c echo.Context) error {

		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return err
		}
		rows, err := db.Query("SELECT username, password FROM user WHERE username = ? AND password = ?", data["username"], data["password"])
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		// Process the query results
		for rows.Next() {
			var username string
			var password string
			if err := rows.Scan(&username, &password); err != nil {
				panic(err.Error())
			}
			fmt.Printf("%s  %s\n", username, password)
			print("Login Successful :)")
		}

		return c.String(http.StatusOK, "Data received successfully")
	})
	e.POST("/upload", uploadHandler)
	e.Logger.Fatal(e.Start(":8080"))
}
