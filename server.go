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
		User:                 "your_username",
		Passwd:               "your_password",
		Net:                  "tcp",
		Addr:                 "localhost:3306",
		DBName:               "your_db_name",
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
	fmt.Print("hi")
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
	client, err := genai.NewClient(ctx, option.WithAPIKey("Your Google API"))
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
		genai.Text("Specifications of Flower in the Image: If the provided image contains a flower, the response will include detailed specifications of the flower, including its attributes such as color, size, scientific name, name, other names, habitat, distribution, etymology, symbolism, uses, and interesting facts. These categories are designed to be understandable to the common user. If the provided image doesn't contain a flower, the response will consist of a single key-value pair: {message: Please provide an image of a flower.} Ensure that the response doesn't contain an empty response in any case and follows a single JSON format without embedding JSON within JSON."),
	}
	res, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return err
	}

	resp, err := printResponse(res)
	if err != nil {
		fmt.Println(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func printResponse(resp *genai.GenerateContentResponse) (map[string]interface{}, error) {
	finalContent := make(map[string]interface{})

	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				// Assume extractPartContent returns a map of attributes
				partContent, err := extractPartContent(part)
				if err != nil {
					return nil, err
				}

				fmt.Println(partContent)

				for key, value := range partContent {
					// Aggregate values in finalContent map.
					finalContent[key] = value
				}
			}
		}
	}

	return finalContent, nil
}

func extractPartContent(part genai.Part) (map[string]interface{}, error) {
	// Marshal the part into JSON
	partJSON, err := json.Marshal(part)
	if err != nil {
		return nil, err
	}

	var jsonString string
	err = json.Unmarshal([]byte(partJSON), &jsonString)
	if err != nil {
		fmt.Println("Error unmarshling to string", err)
	}

	var resultMap map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &resultMap)
	if err != nil {
		fmt.Println("Error unmarshling to map", err)
	}

	return resultMap, nil
}

func main() {

	e := echo.New()
	db := connectDB()

	e.Use(echo.WrapMiddleware(cors.Default().Handler))

	e.GET("/", func(c echo.Context) error {
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
