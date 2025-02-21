package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
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
	ContainerID   string `json:"container_id"`
	Timestamp     string `json:"timestamp"`
	TransactionID string `json:"transaction_id"`
	Version       int    `json:"version"`
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
	txQueue              = make(chan Transaction, 100)
	shardMutexes         [NumShards]sync.Mutex
	retryBackoff         = []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second}
)

// Function to calculate hash for a block
func calculateHash(index int, timestamp string, transactions []Transaction, previousHash string) string {
	txData, _ := json.Marshal(transactions)
	record := fmt.Sprintf("%d%s%s%s", index, timestamp, txData, previousHash)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// Assigns a transaction to a shard based on organization
func getShardID(containerID string) int {
	if len(containerID) < 2 {
		return 0 // Default to shard 0 if ID is too short
	}
	// containerID starts with `org1` or `org2`
	if containerID[:4] == "org1" {
		return 0 // Shard 0 for Org1
	} else if containerID[:4] == "org2" {
		return 1 // Shard 1 for Org2
	}
	// Default fallback: Use hash-based assignment if unknown prefix
	hash := sha256.Sum256([]byte(containerID))
	return int(hash[0]) % NumShards
}

// Validate if block integrity is maintained
func validateBlock(block Block) bool {
	calculatedHash := calculateHash(block.Index, block.Timestamp, block.Transactions, block.PreviousHash)
	return block.Hash == calculatedHash
}

// Function to add a new transaction with concurrency control
func addTransaction(containerID string) error {
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	shardID := getShardID(containerID)

	// Check for concurrency conflicts
	if checkContainerIDExists(containerID) {
		log.Printf("⚠️ Conflict: Transaction exists for %s", containerID)
		logConflict(containerID)
		return fmt.Errorf("conflict: container ID exists")
	}

	timestamp := time.Now().Format(time.RFC3339)
	newTransaction := Transaction{
		ContainerID:   containerID,
		Timestamp:     timestamp,
		TransactionID: fmt.Sprintf("tx-%d", time.Now().UnixNano()),
	}

	if len(Blockchain) > 0 && len(Blockchain[len(Blockchain)-1].Transactions) < maxTransactionsPerBlock {
		latestBlock := &Blockchain[len(Blockchain)-1]

		if latestBlock.ShardID == shardID {
			latestBlock.Transactions = append(latestBlock.Transactions, newTransaction)
			latestBlock.Hash = calculateHash(latestBlock.Index, latestBlock.Timestamp, latestBlock.Transactions, latestBlock.PreviousHash)
			log.Printf("✅ Transaction added to block (Shard %d, Index %d)", shardID, latestBlock.Index)
			return nil
		}
	}
	createNewBlockWithTransaction(newTransaction)
	return nil
}

// logConflict stores concurrency conflicts in a global slice
func logConflict(containerID string) {
	conflictsMu.Lock()
	defer conflictsMu.Unlock()

	conflictMessage := fmt.Sprintf("⚠️ Concurrency conflict detected for container: %s", containerID)

	// Debugging: Log when a conflict is stored
	log.Println("Storing Conflict:", conflictMessage)

	// Store the conflict
	concurrencyConflicts = append(concurrencyConflicts, conflictMessage)

	// Keep only the last 100 conflicts to prevent memory overflow
	if len(concurrencyConflicts) > 100 {
		concurrencyConflicts = concurrencyConflicts[len(concurrencyConflicts)-100:]
	}
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
	shardID := getShardID(containerID) // Ensure shard assignment

	newBlock := Block{
		Index:        len(Blockchain),
		Timestamp:    timestamp,
		ContainerID:  containerID,
		Transactions: []Transaction{},
		PreviousHash: previousHash,
		Hash:         calculateHash(len(Blockchain), timestamp, []Transaction{}, previousHash),
		Version:      version,
		ShardID:      shardID,
	}
	Blockchain = append(Blockchain, newBlock)
	log.Printf("✅ Block added to blockchain (Shard %d): %+v", shardID, newBlock)
}

func createNewBlockWithTransaction(tx Transaction) {
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()
	var previousHash string
	var version int
	// Determine previous block hash and increment version
	if len(Blockchain) > 0 {
		previousHash = Blockchain[len(Blockchain)-1].Hash
		version = Blockchain[len(Blockchain)-1].Version + 1
	} else {
		previousHash = "0" // Genesis block case
		version = 1
	}
	// Assign transaction to a shard based on ContainerID
	shardID := getShardID(tx.ContainerID)
	// Create a new block
	newBlock := Block{
		Index:        len(Blockchain),
		Timestamp:    tx.Timestamp,
		ContainerID:  tx.ContainerID,
		Transactions: []Transaction{tx},
		PreviousHash: previousHash,
		Hash:         calculateHash(len(Blockchain), tx.Timestamp, []Transaction{tx}, previousHash),
		Version:      version,
		ShardID:      shardID,
	}
	Blockchain = append(Blockchain, newBlock)
	log.Printf("✅ New block created (Shard %d, Index %d): %+v", shardID, newBlock.Index, tx)
}

// checkContainerIDExists verifies if a container ID already exists in the blockchain
func checkContainerIDExists(containerID string) bool {
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	// Iterate over all blocks and transactions to check for the container ID
	for _, block := range Blockchain {
		if block.ContainerID == containerID {
			return true
		}
		for _, tx := range block.Transactions {
			if tx.ContainerID == containerID {
				return true
			}
		}
	}
	return false
}

func displayBlockchain() {
	fmt.Println("\n📜 Blockchain:")
	for _, block := range Blockchain {
		fmt.Printf("---------------------------\n")
		fmt.Printf("📌 Index: %d\n", block.Index)
		fmt.Printf("📦 ContainerID: %s\n", block.ContainerID)
		fmt.Printf("🔗 Hash: %s\n", block.Hash)
		fmt.Printf("🔗 Previous Hash: %s\n", block.PreviousHash)
		fmt.Printf("🆕 Version: %d\n", block.Version)
		fmt.Printf("📜 Transactions:\n")
		for _, tx := range block.Transactions {
			fmt.Printf("   - ContainerID: %s | Timestamp: %s | TxID: %s | Version: %d\n", tx.ContainerID, tx.Timestamp, tx.TransactionID, tx.Version)
		}
		fmt.Printf("---------------------------\n")
	}
}
