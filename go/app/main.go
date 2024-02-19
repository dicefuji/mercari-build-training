package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	"path"
	"strings"
	
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
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
}

type ItemsData struct {
	Items [] Item `json:"items"`
}

func appendItemToFile(name string, category string) error {
	file, err := os.OpenFile("items.json", os.O_RDWR|os.O_CREATE, 0755) // Assume file exists
	if err != nil {
		return err
	}
	defer file.Close()

	itemsData := ItemsData{} // Fill with existing data
	err = json.NewDecoder(file).Decode(&itemsData)
	if err != nil && err != io.EOF {
		return err
	}

	newItem := Item{Name: name, Category: category} // Adding  new item
	itemsData.Items = append(itemsData.Items, newItem)

	file.Truncate(0) // Clear file
	file.Seek(0, 0) // Go to start of file
	err = json.NewEncoder(file).Encode(itemsData) // Writing in  new data
	if err != nil {
		return err
	}

	return nil
}
func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category") // i made this change
	c.Logger().Infof("Receive item: %s, Category: %s", name, category)

	err := appendItemToFile(name, category) // Append item to file
	if err != nil {
		c.Logger().Errorf("Error appending item to file: %s", err)
		res := Response{Message: "Error appending item to file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	c.Logger().Infof("Receive item: %s, Category: %s", name, category)
	message := fmt.Sprintf("item received: %s, category: %s", name,category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
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
	e.GET("/image/:imageFilename", getImg)


	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
