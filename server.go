package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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
	client, err := genai.NewClient(ctx, option.WithAPIKey("AIzaSyDE4YvNRYwmkQ4SBrJc4gAzOLXr7Xd6n6Y"))
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
		genai.Text("Provide detailed specifications of flowers in the image provided. Create a JSON format containing attributes like color, size, scientific name, category, other names, habitat, distribution, etymology, symbolism, uses, and interesting facts. Make sure the categories are understandable to the common user, and provide more than 20 categories of specifications."),
	}
	res, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return err
	}

	var jsonResponse map[string]interface{}
	jsonData, err := json.Marshal(res.Candidates[0].Content.Parts[0])
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		// Handle the error accordingly
	} else {
		jsonString := string(jsonData)
		jsonString = strings.TrimSpace(strings.ReplaceAll(jsonString, "\n", ""))
		fmt.Println("JSON string:", jsonString)

		err = json.Unmarshal([]byte(jsonString), &jsonResponse)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			// Handle the error accordingly
		}

		// Use jsonString as needed
	}
	if jsonData == nil {
		return c.String(http.StatusInternalServerError, "Failed to generate content")
	}
	return c.JSON(http.StatusOK, jsonResponse)
}

func printResponse(resp *genai.GenerateContentResponse) []byte {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			test := cand.Content.Parts

			jsonData, err := json.Marshal(test)
			if err != nil {
				fmt.Println("Failed to marshal JSON:", err)
				// TODO handle error return
				return nil
			}

			// Print the JSON data
			return jsonData

		}
	}
	return nil
}

func main() {

	e := echo.New()
	db := connectDB()

	e.Use(echo.WrapMiddleware(cors.Default().Handler))

	e.GET("/", func(c echo.Context) error {
		ctx := context.Background()
		// Access your API key as an environment variable (see "Set up your API key" above)
		client, err := genai.NewClient(ctx, option.WithAPIKey("AIzaSyDE4YvNRYwmkQ4SBrJc4gAzOLXr7Xd6n6Y"))

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
		rowCount := 0
		for rows.Next() {
			var username string
			var password string
			if err := rows.Scan(&username, &password); err != nil {
				panic(err.Error())
			}
			rowCount++
			fmt.Printf("%s  %s\n", username, password)
			print("Login Successful :)")
		}

		if rowCount == 0 {

			return c.String(http.StatusNotFound, "User not found")
		}

		return c.String(http.StatusOK, "Data received successfully")
	})

	e.POST("/reset-password", func(c echo.Context) error {

		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return err
		}

		// New password received from the request
		newPassword := data["password"]

		// Update the password for the specified username
		result, err := db.Exec("UPDATE user SET password = ? WHERE username = ?", newPassword, data["username"])
		if err != nil {
			panic(err.Error())
		}

		// Check the number of rows affected to verify if the update was successful
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			panic(err.Error())
		}

		// If no rows were affected, it means the username doesn't exist
		if rowsAffected == 0 {
			return c.String(http.StatusNotFound, "User not found")
		}

		// If rows were affected, it means the password was updated successfully
		return c.String(http.StatusOK, "Password updated successfully")

	})

	e.POST("/register", func(c echo.Context) error {
		var user User
		if err := c.Bind(&user); err != nil {
			return err
		}
		rows, err := db.Query("SELECT username FROM user WHERE username = ?", user.Username)
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		// Process the query results
		rowCount := 0
		for rows.Next() {
			var username string
			if err := rows.Scan(&username); err != nil {
				panic(err.Error())
			}
			rowCount++
			fmt.Printf("%s \n", username)

		}

		if rowCount != 0 {
			return c.String(http.StatusNotFound, "User already exists Please use other username")
		}

		// Insert user data into the database
		insertRow, err := db.Exec("INSERT INTO user (username, firstName, lastName, email, phoneNumber, password) VALUES (?, ?, ?, ?, ?, ?)",
			user.Username, user.FirstName, user.LastName, user.Email, user.PhoneNumber, user.Password)
		if err != nil {
			panic(err.Error())
		}

		rowsAffected, err := insertRow.RowsAffected()
		if err != nil {
			panic(err.Error())
		}

		if rowsAffected == 0 {
			return c.String(http.StatusNotFound, "Something went wrong")
		}

		return c.String(http.StatusOK, "User registered successfully")
	})

	e.POST("/upload", uploadHandler)
	e.Logger.Fatal(e.Start(":8080"))
}

// User struct for user registration
type User struct {
	Username    string `json:"username"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

type FlowerData struct {
	Color            string   `json:"color"`
	Size             string   `json:"size"`
	ScientificName   string   `json:"scientific_name"`
	Category         string   `json:"category"`
	OtherNames       []string `json:"other_names"`
	Habitat          string   `json:"habitat"`
	Distribution     string   `json:"distribution"`
	Etymology        string   `json:"etymology"`
	Symbolism        string   `json:"symbolism"`
	Uses             []string `json:"uses"`
	InterestingFacts []string `json:"interesting_facts"`
}
