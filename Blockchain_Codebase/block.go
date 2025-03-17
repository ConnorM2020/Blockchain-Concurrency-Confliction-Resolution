package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Blockchain structures
type Block struct {
	Index        int           `json:"index"`
	Timestamp    string        `json:"timestamp"`
	ContainerID  string        `json:"container_id"`
	Transactions []Transaction `json:"transactions"`
	PreviousHash string        `json:"previous_hash"`
	Hash         string        `json:"hash"`
	Version      int           `json:"version"`
	ShardID      int           `json:"shard_id"`
}

// Define Transaction structure
type Transaction struct {
	ContainerID   string  `json:"container_id"`
	Timestamp     string  `json:"timestamp"`
	TransactionID string  `json:"transaction_id"`
	Source        int     `json:"source"`
	Target        int     `json:"target"`
	Version       int     `json:"version"`
	Data          string  `json:"data"`
	Status        string  `json:"status"`
	Type          string  `json:"type"`
	ExecTime      float64 `json:"execTime"`
}

type ShardedTransaction struct {
	Source []int  `json:"source"`
	Target []int  `json:"target"`
	Data   string `json:"data"`
}

// Handling concurrency
type TransactionSegment struct {
	TransactionID string `json:"transaction_id"`
	ShardID       int    `json:"shard_id"`
	SegmentIndex  int    `json:"segment_index"`
	TotalSegments int    `json:"total_segments"`
	Data          string `json:"data"`
}

const (
	NumShards               = 2 // Shard 0 for Org1, Shard 1 for Org2
	maxTransactionsPerBlock = 3 // Limit per block
	maxRetryAttempts        = 3 // Retry attempts for conflicts
)

// Global variables
var (
	concurrencyConflicts []string
	conflictsMu          sync.Mutex
	Blockchain           []Block
	BlockchainMu         sync.Mutex
	transactionMu        sync.Mutex
)

// Function to calculate hash for a block
func calculateHash(index int, timestamp string, transactions []Transaction, previousHash string) string {
	txData, _ := json.Marshal(transactions)
	record := fmt.Sprintf("%d%s%s%s", index, timestamp, txData, previousHash)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// Assigns a transaction to a shard based on organization or fallback hashing
func getShardID(containerID interface{}) int {
	// Ensure containerID is converted to a string
	containerStr := fmt.Sprintf("%v", containerID)

	// Handle short IDs properly
	if len(containerStr) < 1 {
		log.Printf("⚠️ Invalid containerID: '%s' (Too short)", containerStr)
		return 0 // Default to shard 0 if ID is invalid
	}

	// Assign explicit shards for specific organizations
	if strings.HasPrefix(containerStr, "org1") {
		log.Printf("✅ Assigned '%s' to Shard 0 (Org1)", containerStr)
		return 0 // Shard 0 for Org1
	} else if strings.HasPrefix(containerStr, "org2") {
		log.Printf("✅ Assigned '%s' to Shard 1 (Org2)", containerStr)
		return 1
	}
	if numID, err := strconv.Atoi(containerStr); err == nil {
		shard := numID % NumShards
		log.Printf("🔀 Assigned '%s' to Shard %d (Numeric ID)", containerStr, shard)
		return shard
	}
	hash := sha256.Sum256([]byte(containerStr))
	shard := int(hash[0]) % NumShards
	log.Printf("🔀 Assigned '%s' to Shard %d (Fallback Hash)", containerStr, shard)
	return shard
}

func addBlock(containerID string) {
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	var previousHash string
	var version int

	if len(Blockchain) > 0 {
		previousHash = Blockchain[len(Blockchain)-1].Hash
		version = Blockchain[len(Blockchain)-1].Version + 1
	} else {
		previousHash = "0" // Genesis block
		version = 1
	}

	timestamp := time.Now().Format(time.RFC3339)
	shardID := getShardID(containerID) // Assign correct shard

	newBlock := Block{
		Index:        len(Blockchain),
		Timestamp:    timestamp,
		ContainerID:  containerID,
		Transactions: make([]Transaction, 0), // Ensure it's initialized
		PreviousHash: previousHash,
		Hash:         calculateHash(len(Blockchain), timestamp, []Transaction{}, previousHash),
		Version:      version,
		ShardID:      shardID,
	}

	Blockchain = append(Blockchain, newBlock)
	log.Printf("✅ Block added to blockchain (Shard %d): %+v", shardID, newBlock)
}

func addTransactionSegmentHandler(c *gin.Context) {
	var segment TransactionSegment

	if err := c.ShouldBindJSON(&segment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	transactionMu.Lock()
	transactionSegments[segment.TransactionID] = append(transactionSegments[segment.TransactionID], segment)
	transactionMu.Unlock()

	// Check if all segments have arrived
	if len(transactionSegments[segment.TransactionID]) == segment.TotalSegments {
		// Reconstruct full transaction data
		fullData := ""
		for _, seg := range transactionSegments[segment.TransactionID] {
			fullData += seg.Data
		}

		// Ensure blockchain has at least one block
		if len(Blockchain) == 0 {
			log.Println("⚠️ Blockchain is empty, initializing genesis block.")
			addBlock("genesis")
		}

		// Append transaction to the latest block
		newTransaction := Transaction{
			ContainerID:   segment.TransactionID,
			Timestamp:     time.Now().Format(time.RFC3339),
			TransactionID: segment.TransactionID,
			Version:       len(Blockchain),
			Data:          fullData,
			Status:        "completed",
		}

		BlockchainMu.Lock()
		Blockchain[len(Blockchain)-1].Transactions = append(Blockchain[len(Blockchain)-1].Transactions, newTransaction)
		BlockchainMu.Unlock()

		// Log transaction completion
		log.Printf("✅ Transaction fully assembled: %s", fullData)

		// Remove stored transaction segments after successful assembly
		delete(transactionSegments, segment.TransactionID)

		c.JSON(http.StatusOK, gin.H{"message": "Transaction successfully completed"})
		return
	}

	log.Printf("🔄 Received segment %d/%d for transaction %s", segment.SegmentIndex+1, segment.TotalSegments, segment.TransactionID)
	c.JSON(http.StatusOK, gin.H{"message": "Segment received"})
}

func displayBlockchain() {
	fmt.Println("\n📜 Blockchain:")
	for _, block := range Blockchain {
		fmt.Println("---------------------------")
		fmt.Printf("📌 Index: %d\n", block.Index)
		fmt.Printf("📦 ContainerID: %s\n", block.ContainerID)
		fmt.Printf("🔗 Hash: %s\n", block.Hash)
		fmt.Printf("🔗 Previous Hash: %s\n", block.PreviousHash)
		fmt.Printf("🆕 Version: %d\n", block.Version)
		fmt.Printf("📜 Transactions:\n")
		if len(block.Transactions) == 0 {
			fmt.Println("   ❌ No Transactions")
		} else {
			for _, tx := range block.Transactions {
				fmt.Printf("   - TxID: %s | Source: %d → Target: %d | Data: %s | Status: %s\n",
					tx.TransactionID, tx.Source, tx.Target, tx.Data, tx.Status)
			}
		}
		fmt.Println("---------------------------")
	}
}
