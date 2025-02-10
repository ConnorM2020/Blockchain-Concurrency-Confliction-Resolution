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
	numShards               = 4 // Number of shards for concurrency handling
	maxTransactionsPerBlock = 3 // Limit per block
	maxRetryAttempts        = 3 // Retry attempts for conflicts
)

// Global variables
var (
	concurrencyConflicts []string
	conflictsMu          sync.Mutex
	blockchain   []Block
	blockchainMu sync.Mutex
	txQueue      = make(chan Transaction, 100)
	shardMutexes [numShards]sync.Mutex
	retryBackoff = []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second}
)

// Function to calculate hash for a block
func calculateHash(index int, timestamp string, transactions []Transaction, previousHash string) string {
	txData, _ := json.Marshal(transactions)
	record := fmt.Sprintf("%d%s%s%s", index, timestamp, txData, previousHash)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// Assigns a transaction to a shard
func getShardID(containerID string) int {
	hash := sha256.Sum256([]byte(containerID))
	return int(hash[0]) % numShards
}

// Validate if block integrity is maintained
func validateBlock(block Block) bool {
	calculatedHash := calculateHash(block.Index, block.Timestamp, block.Transactions, block.PreviousHash)
	return block.Hash == calculatedHash
}

// Function to add a new transaction with concurrency control
func addTransaction(containerID string) error {
	blockchainMu.Lock()
	defer blockchainMu.Unlock()

	// Check if the containerID already exists in the blockchain
	if checkContainerIDExists(containerID) {
		log.Printf("⚠️ Concurrency conflict detected! Transaction already exists for container: %s", containerID)
		return fmt.Errorf("concurrency conflict: container ID already exists in blockchain")
	}

	timestamp := time.Now().Format(time.RFC3339)
	newTransaction := Transaction{
		ContainerID: containerID,
		Timestamp:   timestamp,
	}

	// Check if latest block has space for a transaction
	if len(blockchain) > 0 && len(blockchain[len(blockchain)-1].Transactions) < maxTransactionsPerBlock {
		latestBlock := &blockchain[len(blockchain)-1]

		latestBlock.Transactions = append(latestBlock.Transactions, newTransaction)
		latestBlock.Hash = calculateHash(latestBlock.Index, latestBlock.Timestamp, latestBlock.Transactions, latestBlock.PreviousHash)

		log.Printf("✅ Transaction added to existing block (Index %d): %+v", latestBlock.Index, newTransaction)
		return nil
	}

	// Create a new block if the latest block is full
	createNewBlockWithTransaction(newTransaction)
	return nil
}

func checkContainerIDExists(containerID string) bool {
	for _, block := range blockchain {
		if block.ContainerID == containerID {
			logConflict(containerID)
			return true
		}
		for _, tx := range block.Transactions {
			if tx.ContainerID == containerID {
				logConflict(containerID)
				return true
			}
		}
	}
	return false
}
func logConflict(containerID string) {
	conflictsMu.Lock()
	defer conflictsMu.Unlock()

	conflictMessage := fmt.Sprintf("⚠️ Concurrency conflict detected for container: %s", containerID)
	concurrencyConflicts = append(concurrencyConflicts, conflictMessage)

	// Keep only the last 100 conflicts to prevent memory overflow
	if len(concurrencyConflicts) > 100 {
		concurrencyConflicts = concurrencyConflicts[len(concurrencyConflicts)-100:]
	}
}


func createNewBlockWithTransaction(tx Transaction) {
	blockchainMu.Lock()
	defer blockchainMu.Unlock()

	var previousHash string
	var version int

	// Determine previous block hash and increment version
	if len(blockchain) > 0 {
		previousHash = blockchain[len(blockchain)-1].Hash
		version = blockchain[len(blockchain)-1].Version + 1
	} else {
		previousHash = "0" // Genesis block case
		version = 1
	}

	// Assign transaction to a shard based on ContainerID
	shardID := getShardID(tx.ContainerID)

	// Create a new block with this transaction
	newBlock := Block{
		Index:        len(blockchain),
		Timestamp:    tx.Timestamp,
		ContainerID:  tx.ContainerID,
		Transactions: []Transaction{tx}, // Add the transaction to the new block
		PreviousHash: previousHash,
		Hash:         calculateHash(len(blockchain), tx.Timestamp, []Transaction{tx}, previousHash),
		Version:      version,
		ShardID:      shardID,
	}

	// Append the new block to the blockchain
	blockchain = append(blockchain, newBlock)
	log.Printf("✅ New block created (Index %d) with transaction: %+v", newBlock.Index, tx)
}

// Function to safely add transaction to blockchain
func addTransactionToBlockchain(tx Transaction, shardID int) bool {
	shardMutexes[shardID].Lock()
	defer shardMutexes[shardID].Unlock()

	blockchainMu.Lock()
	defer blockchainMu.Unlock()

	// Fetch latest block
	var latestBlock *Block
	if len(blockchain) > 0 {
		latestBlock = &blockchain[len(blockchain)-1]
	}

	// Check for version conflicts
	if latestBlock != nil && latestBlock.Version >= tx.Version {
		log.Printf("⚠️ Conflict detected! Current block version: %d, Transaction version: %d", latestBlock.Version, tx.Version)
		return false
	}

	// Add transaction to latest block if space allows
	if latestBlock != nil && len(latestBlock.Transactions) < maxTransactionsPerBlock {
		latestBlock.Transactions = append(latestBlock.Transactions, tx)
		latestBlock.Hash = calculateHash(latestBlock.Index, latestBlock.Timestamp, latestBlock.Transactions, latestBlock.PreviousHash)
		log.Printf("✅ Transaction added to existing block: %+v", latestBlock)
		return true
	}

	// Create a new block if transaction can't fit in the last block
	createNewBlock(tx, shardID)
	return true
}

// Function to create a new block
func createNewBlock(tx Transaction, shardID int) {
	var previousHash string
	var version int

	if len(blockchain) > 0 {
		previousHash = blockchain[len(blockchain)-1].Hash
		version = blockchain[len(blockchain)-1].Version + 1
	} else {
		previousHash = "0" // Genesis block
		version = 1
	}

	newBlock := Block{
		Index:        len(blockchain),
		Timestamp:    tx.Timestamp,
		ContainerID:  tx.ContainerID,
		Transactions: []Transaction{tx},
		PreviousHash: previousHash,
		Hash:         calculateHash(len(blockchain), tx.Timestamp, []Transaction{tx}, previousHash),
		Version:      version,
		ShardID:      shardID,
	}

	blockchain = append(blockchain, newBlock)
	log.Printf("✅ New block created (Block %d): %+v", newBlock.Index, newBlock)
}

// Function to get the latest block's version
func getLatestBlockVersion() int {
	if len(blockchain) == 0 {
		return 0
	}
	return blockchain[len(blockchain)-1].Version
}

// Function to process queued transactions
func processTransactionQueue() {
	for {
		select {
		case tx := <-txQueue:
			shardID := getShardID(tx.ContainerID)
			addTransactionToBlockchain(tx, shardID)
		default:
			time.Sleep(500 * time.Millisecond) // Prevent CPU overutilization
		}
	}
}

// Function to add a block to the blockchain
func addBlock(containerID string) {
	blockchainMu.Lock()
	defer blockchainMu.Unlock()

	var previousHash string
	var version int

	if len(blockchain) > 0 {
		previousHash = blockchain[len(blockchain)-1].Hash
		version = blockchain[len(blockchain)-1].Version + 1
	} else {
		previousHash = "0" // Genesis block
		version = 1
	}

	timestamp := time.Now().Format(time.RFC3339)
	newBlock := Block{
		Index:        len(blockchain),
		Timestamp:    timestamp,
		ContainerID:  containerID,
		Transactions: []Transaction{}, // Empty transaction list
		PreviousHash: previousHash,
		Hash:         calculateHash(len(blockchain), timestamp, []Transaction{}, previousHash),
		Version:      version,
	}

	blockchain = append(blockchain, newBlock)
	log.Printf("✅ Block added to blockchain: %+v", newBlock)
}

// Function to display the blockchain
func displayBlockchain() {
	fmt.Println("\n📜 Blockchain:")
	for _, block := range blockchain {
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
