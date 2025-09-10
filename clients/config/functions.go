package config

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

type ApiConfig struct {
	HttpClient *http.Client
	BaseUrl    string
	ApiKey     string
}

func SendRequest(client *http.Client, apiKey string, body string, url string) (string, error) {
	payload := strings.NewReader(body)

	rq, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	rq.Header.Add("Content-Type", "application/json")
	rq.Header.Add("x-api-key", apiKey)

	resp, err := client.Do(rq)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 409 && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		log.Println("Error sending request: ", resp.Status, " ", string(response))
		return "", fmt.Errorf("error sending request: %s", resp.Status)
	}
	return string(response), nil
}

var client *retryablehttp.Client

func CreateHttpClient() *http.Client {
	if client == nil {
		client = retryablehttp.NewClient()
		client.RetryMax = 3
		client.Logger = nil
	}
	return client.StandardClient()
}
