package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	"path"
	"strings"
	
	"crypto/sha256"
	"encoding/json"

	"strconv"
	"database/sql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "images"
	dbPath = "../db/mercari.sqlite3"
)

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

type Item struct {
	Name string`json:"name"`
	Category string `json:"category"`
	Image string `json:"image"`
}

type Items struct {
	Items [] Item `json:"items"`
}

func appendItemToFile(name string, category string, image string) error {
	file, err := os.OpenFile("items.json", os.O_RDWR|os.O_CREATE, 0755) // Assume file exists
	if err != nil {
		return err
	}
	defer file.Close()

	items := Items{} // Fill with existing data
	err = json.NewDecoder(file).Decode(&items)
	if err != nil && err != io.EOF {
		return err
	}
	
	// this is where we hash the image name
	h := sha256.New()
	h.Write([]byte(strings.Split(image, ".")[0]))
	hashed := h.Sum(nil)
	image = fmt.Sprintf("%x", hashed) + ".jpg"

	newItem := Item{Name: name, Category: category, Image: image} // Add new item

	items.Items = append(items.Items, newItem)

	file.Truncate(0) // Clear file
	file.Seek(0, 0) // Go to start of file
	err = json.NewEncoder(file).Encode(items) // Writing in  new data
	if err != nil {
		return err
	}

	return nil
}
func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category") // i made this change
	image := c.FormValue("image")

	err := appendItemToFile(name, category,image) // Append item to file
	if err != nil {
		c.Logger().Errorf("Error appending item to file: %s", err)
		res := Response{Message: "Error appending item to file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	c.Logger().Infof("Receive item: %s, Category: %s, Image: %s", name, category, image)
	message := fmt.Sprintf("item received: %s, category: %s", name,category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	file, err := os.Open("items.json") // Open file
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	defer file.Close() 

	items := Items{} // Fill with existing data
	err = json.NewDecoder(file).Decode(&itemsData)
	if err != nil && err != io.EOF {
		c.Logger().Errorf("Error decoding file: %s", err)
		res := Response{Message: "Error decoding file"}
		return c.JSON(http.StatusInternalServerError, res)
	}

	return c.JSON(http.StatusOK, items)
}

func getItemById(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id")) // Get item id
	file, err := os.Open("items.json") // Open file
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	defer file.Close()

	items := Items{} // Fill with existing data
	err = json.NewDecoder(file).Decode(&items)
	if err != nil && err != io.EOF {
		c.Logger().Errorf("Error decoding file: %s", err)
		res := Response{Message: "Error decoding file"}
		return c.JSON(http.StatusInternalServerError, res)
	}

	return c.JSON(http.StatusOK, items.Items[id-1])
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItems)
	e.GET("/items/:id", getItemById)
	e.GET("/image/:imageFilename", getImg)


	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
