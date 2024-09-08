package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/labstack/echo/v4"
)

const SERVICEPORT = 1325
const BROKER = "pubsub-redis"
const TOPIC = "products"

type Subscription struct {
	PubSubName string `json:"pubsubname"`
	Topic      string `json:"topic"`
	Route      string `json:"route"`
}

type Product struct {
	ProductID         int       `json:"product_id"`
	ProductName       string    `json:"product_name"`
	ProductTargetTime time.Time `json:"product_target_time"`
	CreatedAt         time.Time `json:"created_at"`
}

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "This is the ERP service!")
	})
	e.GET("/dapr/subscribe", subscribe)
	e.POST("/monitor", handleSubscriptions)

	e.Logger.Fatal(e.Start(fmt.Sprintf("localhost:%v", SERVICEPORT)))
}

/**
 * Subscription endpoint for Dapr.
 */
func subscribe(c echo.Context) error {
	subscriptions := []Subscription{
		{
			PubSubName: BROKER,
			Topic:      TOPIC,
			Route:      "monitor",
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

	checkProduct(receivedProduct)

	return c.NoContent(http.StatusNoContent)
}

/**
 * Check product creation with master data.
 */
func checkProduct(p Product) {
	resp := invokeService("erp", "products/"+strconv.Itoa(p.ProductID))

	var masterData Product

	err := json.NewDecoder(resp.Body).Decode(&masterData)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Message received - ProductId: %d, Created At: %s\n", p.ProductID, p.CreatedAt)

	if masterData.ProductTargetTime.After(p.CreatedAt) || masterData.ProductTargetTime.Equal(p.CreatedAt) {
		fmt.Printf("Production is in time for %s (TargetTime: %s)\n", masterData.ProductName, masterData.ProductTargetTime)
	} else {
		fmt.Printf("Delay for %s detected! (TargetTime: %s)\n", masterData.ProductName, masterData.ProductTargetTime)
	}
}

/**
 * Invoke a service method over Dapr.
 */
func invokeService(service string, method string) (resp *http.Response) {
	daprHost := os.Getenv("DAPR_HOST")
	if daprHost == "" {
		daprHost = "http://localhost"
	}

	daprHttpPort := os.Getenv("DAPR_HTTP_PORT")
	if daprHttpPort == "" {
		daprHttpPort = "3500"
	}

	daprURL := daprHost + ":" + daprHttpPort + "/v1.0/invoke/" + service + "/method/" + method
	token := getDaprAPIToken()

	client := &http.Client{}

	req, err := http.NewRequest("GET", daprURL, nil)
	if err != nil {
		panic(err.Error())
	}

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
 * Get the set API token.
 */
func getDaprAPIToken() (token string) {
	return os.Getenv("DAPR_API_TOKEN")
}
