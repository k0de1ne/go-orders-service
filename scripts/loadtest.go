package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestConfig struct {
	BaseURL       string
	TotalRequests int
	Concurrency   int
	Duration      time.Duration
}

type Stats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    int64
	MinLatency      int64
	MaxLatency      int64
	Errors          sync.Map
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Service base URL")
	requests := flag.Int("requests", 1000, "Total number of requests")
	concurrency := flag.Int("concurrency", 10, "Number of parallel requests")
	duration := flag.Duration("duration", 0, "Test duration (0 = use -requests)")
	operation := flag.String("operation", "create", "Operation type: create, get, update, delete, list, mixed")
	flag.Parse()

	config := LoadTestConfig{
		BaseURL:       *baseURL,
		TotalRequests: *requests,
		Concurrency:   *concurrency,
		Duration:      *duration,
	}

	fmt.Printf("🚀 Starting load test\n")
	fmt.Printf("URL: %s\n", config.BaseURL)
	fmt.Printf("Operation: %s\n", *operation)
	if config.Duration > 0 {
		fmt.Printf("Duration: %v\n", config.Duration)
	} else {
		fmt.Printf("Requests: %d\n", config.TotalRequests)
	}
	fmt.Printf("Concurrency: %d\n\n", config.Concurrency)

	stats := &Stats{
		MinLatency: int64(^uint64(0) >> 1), // max int64
	}

	startTime := time.Now()

	switch *operation {
	case "create":
		runCreateTest(config, stats)
	case "get":
		runGetTest(config, stats)
	case "list":
		runListTest(config, stats)
	case "update":
		runUpdateTest(config, stats)
	case "delete":
		runDeleteTest(config, stats)
	case "mixed":
		runMixedTest(config, stats)
	default:
		fmt.Printf("Unknown operation: %s\n", *operation)
		return
	}

	elapsed := time.Since(startTime)

	printResults(stats, elapsed)
}

func runCreateTest(config LoadTestConfig, stats *Stats) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)
	endTime := time.Now().Add(config.Duration)

	for (config.Duration <= 0 || !time.Now().After(endTime)) &&
		(config.Duration != 0 || requestCount < int64(config.TotalRequests)) {
		wg.Add(1)
		semaphore <- struct{}{}
		atomic.AddInt64(&requestCount, 1)

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			createOrder(config.BaseURL, stats)
		}()
	}

	wg.Wait()
}

func runGetTest(config LoadTestConfig, stats *Stats) {
	orderIDs := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		id := createOrderAndGetID(config.BaseURL)
		if id != "" {
			orderIDs = append(orderIDs, id)
		}
	}

	if len(orderIDs) == 0 {
		fmt.Println("❌ Failed to create orders for test")
		return
	}

	fmt.Printf("✅ Created %d orders for testing\n\n", len(orderIDs))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)
	endTime := time.Now().Add(config.Duration)

	for (config.Duration <= 0 || !time.Now().After(endTime)) &&
		(config.Duration != 0 || requestCount < int64(config.TotalRequests)) {
		wg.Add(1)
		semaphore <- struct{}{}
		idx := atomic.AddInt64(&requestCount, 1)

		go func(index int64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			orderID := orderIDs[index%int64(len(orderIDs))]
			getOrder(config.BaseURL, orderID, stats)
		}(idx)
	}

	wg.Wait()
}

func runListTest(config LoadTestConfig, stats *Stats) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)
	endTime := time.Now().Add(config.Duration)

	for (config.Duration <= 0 || !time.Now().After(endTime)) &&
		(config.Duration != 0 || requestCount < int64(config.TotalRequests)) {
		wg.Add(1)
		semaphore <- struct{}{}
		atomic.AddInt64(&requestCount, 1)

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			listOrders(config.BaseURL, stats)
		}()
	}

	wg.Wait()
}

func runMixedTest(config LoadTestConfig, stats *Stats) {
	orderIDs := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		id := createOrderAndGetID(config.BaseURL)
		if id != "" {
			orderIDs = append(orderIDs, id)
		}
	}

	fmt.Printf("✅ Created %d orders for mixed test\n\n", len(orderIDs))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)
	endTime := time.Now().Add(config.Duration)

	for (config.Duration <= 0 || !time.Now().After(endTime)) &&
		(config.Duration != 0 || requestCount < int64(config.TotalRequests)) {
		wg.Add(1)
		semaphore <- struct{}{}
		idx := atomic.AddInt64(&requestCount, 1)

		go func(index int64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			op := index % 10
			switch {
			case op < 4:
				createOrder(config.BaseURL, stats)
			case op < 7:
				if len(orderIDs) > 0 {
					orderID := orderIDs[index%int64(len(orderIDs))]
					getOrder(config.BaseURL, orderID, stats)
				}
			case op < 9:
				listOrders(config.BaseURL, stats)
			default:
				if len(orderIDs) > 0 {
					orderID := orderIDs[index%int64(len(orderIDs))]
					updateOrder(config.BaseURL, orderID, stats)
				}
			}
		}(idx)
	}

	wg.Wait()
}

func createOrder(baseURL string, stats *Stats) string {
	payload := map[string]interface{}{
		"product":  fmt.Sprintf("Product-%d", time.Now().UnixNano()),
		"quantity": 10,
	}

	return makeRequest("POST", baseURL+"/orders", payload, stats)
}

func createOrderAndGetID(baseURL string) string {
	payload := map[string]interface{}{
		"product":  fmt.Sprintf("Product-%d", time.Now().UnixNano()),
		"quantity": 10,
	}

	body := makeRequestRaw("POST", baseURL+"/orders", payload)
	if body == "" {
		return ""
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return ""
	}

	if id, ok := result["id"].(string); ok {
		return id
	}
	return ""
}

func getOrder(baseURL, orderID string, stats *Stats) string {
	return makeRequest("GET", baseURL+"/orders/"+orderID, nil, stats)
}

func listOrders(baseURL string, stats *Stats) string {
	return makeRequest("GET", baseURL+"/orders", nil, stats)
}

func updateOrder(baseURL, orderID string, stats *Stats) string {
	payload := map[string]interface{}{
		"product":  fmt.Sprintf("Updated-%d", time.Now().UnixNano()),
		"quantity": 20,
		"status":   "confirmed",
	}

	return makeRequest("PUT", baseURL+"/orders/"+orderID, payload, stats)
}

func makeRequest(method, url string, payload interface{}, stats *Stats) string {
	start := time.Now()
	atomic.AddInt64(&stats.TotalRequests, 1)

	var reqBody io.Reader
	if payload != nil {
		jsonData, _ := json.Marshal(payload)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		recordError(stats, err)
		return ""
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		recordError(stats, err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	latency := time.Since(start).Milliseconds()
	recordLatency(stats, latency)

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddInt64(&stats.SuccessRequests, 1)
		return string(body)
	} else {
		atomic.AddInt64(&stats.FailedRequests, 1)
		recordError(stats, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
		return ""
	}
}

func makeRequestRaw(method, url string, payload interface{}) string {
	var reqBody io.Reader
	if payload != nil {
		jsonData, _ := json.Marshal(payload)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return ""
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func recordLatency(stats *Stats, latency int64) {
	atomic.AddInt64(&stats.TotalLatency, latency)

	for {
		old := atomic.LoadInt64(&stats.MinLatency)
		if latency >= old {
			break
		}
		if atomic.CompareAndSwapInt64(&stats.MinLatency, old, latency) {
			break
		}
	}

	for {
		old := atomic.LoadInt64(&stats.MaxLatency)
		if latency <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&stats.MaxLatency, old, latency) {
			break
		}
	}
}

func recordError(stats *Stats, err error) {
	atomic.AddInt64(&stats.FailedRequests, 1)
	errMsg := err.Error()
	val, _ := stats.Errors.LoadOrStore(errMsg, new(int64))
	atomic.AddInt64(val.(*int64), 1)
}

func printResults(stats *Stats, elapsed time.Duration) {
	total := atomic.LoadInt64(&stats.TotalRequests)
	success := atomic.LoadInt64(&stats.SuccessRequests)
	failed := atomic.LoadInt64(&stats.FailedRequests)
	totalLatency := atomic.LoadInt64(&stats.TotalLatency)
	minLatency := atomic.LoadInt64(&stats.MinLatency)
	maxLatency := atomic.LoadInt64(&stats.MaxLatency)

	fmt.Printf("\n📊 Load Test Results\n")
	fmt.Printf("═══════════════════════════════════════════════════\n")
	fmt.Printf("Total time:           %v\n", elapsed)
	fmt.Printf("Total requests:       %d\n", total)
	fmt.Printf("Successful:           %d (%.2f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Failed:               %d (%.2f%%)\n", failed, float64(failed)/float64(total)*100)
	fmt.Printf("\n")
	fmt.Printf("Throughput:           %.2f req/sec\n", float64(total)/elapsed.Seconds())
	fmt.Printf("\n")
	fmt.Printf("Latency:\n")
	fmt.Printf("  Average:            %d ms\n", totalLatency/total)
	fmt.Printf("  Minimum:            %d ms\n", minLatency)
	fmt.Printf("  Maximum:            %d ms\n", maxLatency)

	if failed > 0 {
		fmt.Printf("\n❌ Errors:\n")
		stats.Errors.Range(func(key, value interface{}) bool {
			count := atomic.LoadInt64(value.(*int64))
			fmt.Printf("  [%d] %s\n", count, key.(string))
			return true
		})
	}
	fmt.Printf("═══════════════════════════════════════════════════\n")
}

func runUpdateTest(config LoadTestConfig, stats *Stats) {
	orderIDs := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		id := createOrderAndGetID(config.BaseURL)
		if id != "" {
			orderIDs = append(orderIDs, id)
		}
	}

	fmt.Printf("✅ Created %d orders for update\n\n", len(orderIDs))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)
	endTime := time.Now().Add(config.Duration)

	for (config.Duration <= 0 || !time.Now().After(endTime)) &&
		(config.Duration != 0 || requestCount < int64(config.TotalRequests)) {
		wg.Add(1)
		semaphore <- struct{}{}
		idx := atomic.AddInt64(&requestCount, 1)

		go func(index int64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			orderID := orderIDs[index%int64(len(orderIDs))]
			updateOrder(config.BaseURL, orderID, stats)
		}(idx)
	}

	wg.Wait()
}

func runDeleteTest(config LoadTestConfig, stats *Stats) {
	fmt.Println("⚠️  Delete test requires creating orders before deletion...")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	requestCount := int64(0)

	for requestCount < int64(config.TotalRequests) {
		wg.Add(1)
		semaphore <- struct{}{}
		atomic.AddInt64(&requestCount, 1)

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			id := createOrderAndGetID(config.BaseURL)
			if id != "" {
				deleteOrder(config.BaseURL, id, stats)
			} else {
				atomic.AddInt64(&stats.FailedRequests, 1)
			}
		}()
	}

	wg.Wait()
}

func deleteOrder(baseURL, orderID string, stats *Stats) string {
	return makeRequest("DELETE", baseURL+"/orders/"+orderID, nil, stats)
}
