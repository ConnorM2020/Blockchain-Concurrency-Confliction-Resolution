package blockchain_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// API Base URL
const API_BASE = "http://localhost:8080"

// Transaction represents a blockchain transaction
type Transaction struct {
	Source int    `json:"source"`
	Target int    `json:"target"`
	Data   string `json:"data"`
}

// GenerateLargeData creates a random string of given size
func GenerateLargeData(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// SendTransaction sends a transaction via HTTP POST
// SendTransaction sends a transaction via HTTP POST
func SendTransaction(wg *sync.WaitGroup, isSharded bool) {
	if wg != nil {
		defer wg.Done()
	}

	tx := Transaction{
		Source: rand.Intn(5) + 1, // Random source (1-5)
		Target: rand.Intn(5) + 6, // Random target (6-10)
		Data:   GenerateLargeData(50),
	}

	jsonData, err := json.Marshal(tx)
	if err != nil {
		fmt.Println("❌ Error marshaling JSON:", err)
		return
	}

	// ✅ Select the correct API endpoint
	endpoint := "/addTransaction" // Default: Non-Sharded API
	if isSharded {
		endpoint = "/addShardedTransaction" // Use the Sharded API when necessary
	}

	resp, err := http.Post(API_BASE+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("❌ Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		fmt.Printf("✅ Transaction Sent Successfully to %s\n", endpoint)
	} else {
		fmt.Printf("❌ Error: Received status code %d from %s\n", resp.StatusCode, endpoint)
	}
}

// ProcessNonSharded executes transactions sequentially
func ProcessNonSharded(numTransactions int) {
	start := time.Now()
	for i := 0; i < numTransactions; i++ {
		SendTransaction(nil, false)
	}
	duration := time.Since(start)
	fmt.Printf("⏳ Non-Sharded Execution Time: %v\n", duration)
}

// ProcessSharded executes transactions in parallel
func ProcessSharded(numTransactions, numShards int) {
	fmt.Println("⚡ [DEBUG] ProcessSharded() was called with transactions:", numTransactions, "shards:", numShards)
	start := time.Now()
	var wg sync.WaitGroup
	shardSize := numTransactions / numShards

	for i := 0; i < numShards; i++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()
			for j := 0; j < shardSize; j++ {
				SendTransaction(nil, true)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("⚡ Sharded Execution Time (%d shards): %v\n", numShards, duration)
}

// RunStressTest executes parallel transactions
func RunStressTest() {
	start := time.Now()
	var wg sync.WaitGroup
	numTransactions := 10

	fmt.Println("🚀 Running Blockchain Stress Test...")

	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go SendTransaction(&wg, false)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("✅ Stress Test Completed in %v\n", duration)
}
