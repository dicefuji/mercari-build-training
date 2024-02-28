package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	"path"
	"strings"
	
	"crypto/sha256"
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
	Message string
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

type Item struct {
	Id int
	Name string 
	Category string 
	ImageName string 
}

type Items struct {
	Items []*Item
}

func hashName(name string) string {
	// this is where we hash the image name
	

	hash := sha256.New()
	hash.Write([]byte(name))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
	

func addItem(c echo.Context) error {
    // Get form data
    name := c.FormValue("name")
    category := c.FormValue("category")
	src, _ := c.FormFile("image_name")

	// Open file for reading
	srcFile, err := src.Open()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer srcFile.Close()

	image_name := hashName(strings.Split(src.Filename, ".")[0]) + ".jpg"

	// Create image file
	dstFile, err := os.Create(path.Join(ImgDir, image_name))
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer dstFile.Close()

	// Copy image to file
	io.Copy(dstFile, srcFile)
    c.Logger().Infof("Received item: %s, %s, %s", name, category, image_name)
    message := fmt.Sprintf("Item received: %s, %s, %s", name, category, image_name)

    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }
    defer db.Close()

    // Insert category
    stmnt1, err := db.Prepare(
        "INSERT OR IGNORE INTO category (name) VALUES (?)",
    )
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }
    defer stmnt1.Close()

    _, err = stmnt1.Exec(category)
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }

    // Get category ID
    var categoryID int
    err = db.QueryRow("SELECT id FROM category WHERE name = ?", category).Scan(&categoryID)
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }

    // Insert item
    stmnt2, err := db.Prepare(
        "INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)",
    )
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }
    defer stmnt2.Close()

    _, err = stmnt2.Exec(name, categoryID, image_name)
    if err != nil {
        return echo.NewHTTPError(
            http.StatusInternalServerError,
            err.Error(),
        )
    }

    res := Response{Message: message}
    return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer db.Close() 

	// Get all items from db
	rows, err := db.Query(
		"SELECT i.name, c.name, i.image_name FROM items AS i JOIN category AS c ON i.category_id = c.id",
	)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer rows.Close()

	// Map items to returned var
	items := Items{Items: []*Item{}}
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.Name,
			&item.Category,
			&item.ImageName,
		); if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				err.Error(),
			)
		}
		items.Items = append(items.Items, &item)
	}

	return c.JSON(http.StatusOK, items)
}

func searchItems(c echo.Context) error {
	// Get search keyword query param
	keyword := c.QueryParam("keyword")
	c.Logger().Infof("Searching for: %s", keyword)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer db.Close()

	// Search for items
	rows, err := db.Query(
		"SELECT i.name, c.name, image_name FROM items AS i JOIN category AS c ON i.category_id = c.id WHERE i.name LIKE ? OR c.name LIKE ?",
		"%"+keyword+"%",
		"%"+keyword+"%",
	)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}
	defer rows.Close()

	// Map items to returned var
	items := Items{Items: []*Item{}}
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.Name,
			&item.Category,
			&item.ImageName,
		)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				err.Error(),
			)
		}
		items.Items = append(items.Items, &item)
	}
	
	return c.JSON(http.StatusOK, items)
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
	e.GET("/search", searchItems)
	e.GET("/image/:imageFilename", getImg)


	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
