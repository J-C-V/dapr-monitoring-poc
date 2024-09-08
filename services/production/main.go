package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const MAXPRODUCTID = 2
const BROKER = "pubsub-redis"
const TOPIC = "products"

type Product struct {
	ProductID int       `json:"product_id"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	daprURL := getDaprURL()
	contentType := "application/json"
	token := getDaprAPIToken()

	for {
		rng := rand.Intn(MAXPRODUCTID)

		var product = Product{
			ProductID: (rng + 1),
			CreatedAt: getRandomTimestamp(),
		}

		data, err := json.Marshal(product)
		if err != nil {
			panic(err.Error())
		}

		client := &http.Client{}

		req, err := http.NewRequest("POST", daprURL, bytes.NewBuffer(data))
		if err != nil {
			panic(err.Error())
		}

		req.Header.Add("Content-Type", contentType)
		req.Header.Add("dapr-api-token", token)

		_, err = client.Do(req)
		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Published new data for ProductId %d\n", product.ProductID)

		time.Sleep(time.Second * time.Duration(5))
	}
}

/**
 * Get URL of Dapr sidecars Publish/Subscribe building block.
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

	return daprHost + ":" + daprHttpPort + "/v1.0/publish/" + BROKER + "/" + TOPIC
}

/**
 * Get the set API token.
 */
func getDaprAPIToken() (token string) {
	return os.Getenv("DAPR_API_TOKEN")
}

/**
 * Get a random timestamp of the current day.
 */
func getRandomTimestamp() time.Time {
	now := time.Now()

	randomHours := rand.Intn(24)
	randomMinutes := rand.Intn(60)
	randomSeconds := rand.Intn(60)

	return time.Date(
		now.Year(), now.Month(), now.Day(),
		randomHours, randomMinutes, randomSeconds, 0, now.Location(),
	)
}
