package stress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// API Base URL
const API_BASE = "http://localhost:8080"

// Function to generate large random data
func generateLargeData(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < size; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// Function to send a single transaction
func sendTransaction(wg *sync.WaitGroup) {
	defer wg.Done()

	// Construct payload
	payload := map[string]interface{}{
		"source": rand.Intn(5) + 1, // Random source (1-5)
		"target": rand.Intn(5) + 6, // Random target (6-10)
		"data":   generateLargeData(50),
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("❌ Error marshalling JSON:", err)
		return
	}

	// Send HTTP POST request
	resp, err := http.Post(API_BASE+"/addTransaction", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("❌ Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusAccepted {
		fmt.Println("Transaction Sent Successfully!")
	} else {
		fmt.Printf("Error: Received status code %d\n", resp.StatusCode)
	}
}

// Main function re-named
func RunTransactionTest() {
	rand.Seed(time.Now().UnixNano()) // Seed random generator

	var wg sync.WaitGroup
	numTransactions := 10 // Number of parallel transactions

	fmt.Println("🚀 Sending transactions...")

	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go sendTransaction(&wg) // Start goroutine
	}

	wg.Wait() // Wait for all transactions to complete
	fmt.Println("✅ All transactions processed.")
}
