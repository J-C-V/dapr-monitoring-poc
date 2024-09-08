package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/labstack/echo/v4"
)

const SERVICEPORT = 1324
const BROKER = "pubsub-redis"
const TOPIC = "products"
const BINDING = "bindings-postgres"

type Subscription struct {
	PubSubName string `json:"pubsubname"`
	Topic      string `json:"topic"`
	Route      string `json:"route"`
}

type SQLMetadata interface{}

type SQLOperation struct {
	Operation string                 `json:"operation"`
	Metadata  SQLMetadata            `json:"metadata"`
	Data      map[string]interface{} `json:"data"`
}

type SQLRequestMetadata struct {
	SQL    string `json:"sql"`
	Params string `json:"params"`
}

type Product struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	initTable()

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "This is the Data service!")
	})
	e.GET("/products", getProducts)
	e.GET("/dapr/subscribe", subscribe)
	e.POST("/store", handleSubscriptions)

	e.Logger.Fatal(e.Start(fmt.Sprintf("localhost:%v", SERVICEPORT)))
}

/**
 * Get URL of Dapr sidecars Binding building block.
 */
func getDaprURL() (url string) {
	daprHost := os.Getenv("DAPR_HOST")
	if daprHost == "" {
		daprHost = "http://localhost"
	}

	daprHttpPort := os.Getenv("DAPR_HTTP_PORT")
	if daprHttpPort == "" {
		daprHttpPort = "3500"
	}

	return daprHost + ":" + daprHttpPort + "/v1.0/bindings/" + BINDING
}

/**
 * Get the set API token.
 */
func getDaprAPIToken() (token string) {
	return os.Getenv("DAPR_API_TOKEN")
}

/**
 * Request a SQL operation over Dapr.
 */
func requestSQLOperation(sqlCmd SQLOperation) (resp *http.Response) {
	daprURL := getDaprURL()
	contentType := "application/json"
	token := getDaprAPIToken()

	payload, err := json.Marshal(sqlCmd)
	if err != nil {
		panic(err.Error())
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", daprURL, bytes.NewBuffer(payload))
	if err != nil {
		panic(err.Error())
	}

	req.Header.Add("Content-Type", contentType)
	req.Header.Add("dapr-api-token", token)

	resp, err = client.Do(req)
	if err != nil {
		panic(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		panic(string(body))
	}

	return resp
}

/**
 * Initialize test table.
 */
func initTable() {
	sqlCmd := SQLOperation{
		Operation: "exec",
		Metadata: SQLRequestMetadata{
			SQL: `CREATE TABLE IF NOT EXISTS products (
				id SERIAL PRIMARY KEY,
				product_id int,
				created_at TIMESTAMP
			);`,
		},
	}

	requestSQLOperation(sqlCmd)
	fmt.Println("Initialized database successfully")
}

/**
 * Subscription endpoint for Dapr.
 */
func subscribe(c echo.Context) error {
	subscriptions := []Subscription{
		{
			PubSubName: BROKER,
			Topic:      TOPIC,
			Route:      "store",
		},
	}

	return c.JSON(http.StatusOK, subscriptions)
}

/**
 * Incoming message handler for Dapr.
 */
func handleSubscriptions(c echo.Context) error {
	event := cloudevents.NewEvent()
	err := json.NewDecoder(c.Request().Body).Decode(&event)
	if err != nil {
		panic(err.Error())
	}

	var receivedProduct Product

	err = json.Unmarshal(event.Data(), &receivedProduct)
	if err != nil {
		panic(err.Error())
	}

	storeProduct(receivedProduct)
	fmt.Printf("Message received for ProductId %d!\n", receivedProduct.ProductID)

	return c.NoContent(http.StatusNoContent)
}

/**
 * Store a product.
 */
func storeProduct(product Product) {
	sqlCmd := SQLOperation{
		Operation: "exec",
		Metadata: SQLRequestMetadata{
			SQL: `INSERT INTO products (product_id, created_at)
			      VALUES ($1, $2);`,
			Params: fmt.Sprintf("[%v, \"%v\"]", product.ProductID, product.CreatedAt.Format(time.RFC3339)),
		},
	}

	requestSQLOperation(sqlCmd)
}

/**
 * Get the last 10 created products.
 */
func getProducts(c echo.Context) error {
	query := "SELECT * FROM products ORDER BY id DESC LIMIT 10;"

	sqlCmd := SQLOperation{
		Operation: "query",
		Metadata: SQLRequestMetadata{
			SQL: query,
		},
	}

	resp := requestSQLOperation(sqlCmd)
	defer resp.Body.Close()

	var respData [][]interface{}
	err := json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		panic(err.Error())
	}

	var products []Product
	for _, data := range respData {
		parsedTime, err := time.Parse(time.RFC3339, data[2].(string))
		if err != nil {
			panic(err.Error())
		}

		product := Product{
			ID:        int(data[0].(float64)),
			ProductID: int(data[1].(float64)),
			CreatedAt: parsedTime,
		}

		products = append(products, product)
	}

	return c.JSON(http.StatusOK, products)
}
