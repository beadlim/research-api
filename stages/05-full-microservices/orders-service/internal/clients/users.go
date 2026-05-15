package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var usersCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "inter_service_call_duration_seconds",
	Help:    "Latency of HTTP calls between services",
	Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
}, []string{"caller", "target", "endpoint", "status"})

type UsersClient struct {
	baseURL string
	http    *http.Client
}

func NewUsersClient(baseURL string) *UsersClient {
	return &UsersClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *UsersClient) Exists(ctx context.Context, userID int) (bool, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/users/%d", c.baseURL, userID), nil)
	if err != nil {
		return false, err
	}

	resp, err := c.http.Do(req)
	elapsed := time.Since(start).Seconds()

	status := "error"
	if err == nil {
		status = fmt.Sprintf("%d", resp.StatusCode)
		resp.Body.Close()
	}
	usersCallDuration.WithLabelValues("orders-service", "users-service", "/users/{id}", status).Observe(elapsed)

	if err != nil {
		return false, err
	}
	return resp.StatusCode == http.StatusOK, nil
}

type ProductPrice struct {
	ID    int     `json:"id"`
	Price float64 `json:"price"`
}

type ProductsClient struct {
	baseURL string
	http    *http.Client
}

func NewProductsClient(baseURL string) *ProductsClient {
	return &ProductsClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ProductsClient) GetPrice(ctx context.Context, productID int) (float64, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/products/%d", c.baseURL, productID), nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.http.Do(req)
	elapsed := time.Since(start).Seconds()

	status := "error"
	if err == nil {
		status = fmt.Sprintf("%d", resp.StatusCode)
	}
	usersCallDuration.WithLabelValues("orders-service", "products-service", "/products/{id}", status).Observe(elapsed)

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("product %d not found", productID)
	}

	var p ProductPrice
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return 0, err
	}
	return p.Price, nil
}
