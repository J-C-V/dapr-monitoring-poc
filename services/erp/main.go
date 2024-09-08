package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

const SERVICEPORT = 1323

type Product struct {
	ProductID         int       `json:"product_id"`
	ProductName       string    `json:"product_name"`
	ProductTargetTime time.Time `json:"product_target_time"`
}

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "This is the ERP service!")
	})
	e.GET("/products", getProducts)
	e.GET("/products/:id", getProduct)

	e.Logger.Fatal(e.Start(fmt.Sprintf("0.0.0.0:%v", SERVICEPORT)))
}

/**
 * Get product store.
 */
func getProductsStore() []Product {
	now := time.Now()

	return []Product{
		{
			ProductID:         1,
			ProductName:       "Product A",
			ProductTargetTime: time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()),
		},
		{
			ProductID:         2,
			ProductName:       "Product B",
			ProductTargetTime: time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location()),
		},
	}
}

/**
 * Get products.
 */
func getProducts(c echo.Context) error {
	products := getProductsStore()

	return c.JSON(http.StatusOK, products)
}

/**
 * Get specific product.
 */
func getProduct(c echo.Context) error {
	param := c.Param("id")

	id, err := strconv.Atoi(param)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid product id, must be an integer.",
		})
	}

	products := getProductsStore()

	if id < 1 || id > len(products) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Product not found.",
		})
	}

	selectedProduct := products[id-1]

	return c.JSON(http.StatusOK, selectedProduct)
}
