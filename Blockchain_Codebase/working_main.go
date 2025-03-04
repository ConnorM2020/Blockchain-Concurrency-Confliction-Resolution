package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

const (
	blockchainFile = "blockchain.json"
	maxRetries     = 3
)

var (
	port    string
	process bool
	server  bool

	transactionStatus = make(map[string]string)
)

// Middleware to allow CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Initialize CLI flags
func init() {
	flag.StringVar(&port, "port", "8080", "Port number to run the server")
	flag.BoolVar(&process, "process", false, "Process containers and add to blockchain")
	flag.BoolVar(&server, "server", false, "Run REST API server for inspecting containers")
	flag.Parse()

	initShards() // Ensure sharding system is initialized
}

// Process running Docker containers and add them to the blockchain
func processContainers(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("❌ Error listing containers: %v", err)
	}

	if len(containers) == 0 {
		log.Println("⚠️ No active containers found.")
		return
	}

	log.Println("📦 Processing running Docker Containers...")
	var wg sync.WaitGroup

	for _, container := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			log.Printf("🛠️ Processing container: %s", containerID)
			addBlock(containerID)
		}(container.ID)
	}

	wg.Wait()
	log.Println("✅ Finished processing all containers.")

	// Display Blockchain and Shard Distribution
	displayBlockchain()
	visualizeShards()
}

func checkTransactionStatus(c *gin.Context) {
	transactionID := c.Param("transactionID")

	// First, check in the transactionStatus map
	transactionMu.Lock()
	status, exists := transactionStatus[transactionID]
	transactionMu.Unlock()

	if exists {
		c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": status})
		return
	}
	// If not found in pending, check in the blockchain itself
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	for _, block := range Blockchain {
		for _, tx := range block.Transactions {
			if tx.TransactionID == transactionID {
				// Mark transaction as completed in transactionStatus
				transactionMu.Lock()
				transactionStatus[transactionID] = "completed"
				transactionMu.Unlock()

				c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": "completed", "data": tx.Data})
				return
			}
		}
	}
	// If still not found, return pending
	c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": "pending"})
}

// Start the REST API server
func runAPIServer(cli *client.Client) {
	r := gin.Default()
	r.Use(CORSMiddleware()) // Enable CORS
	// API Endpoints
	r.GET("/blockchain", getBlockchain)
	r.GET("/blockchain/shard", getShardBlockchain)
	r.GET("/transactionStatus/:transactionID", checkTransactionStatus)

	r.POST("/resetBlockchain", resetBlockchainHandler)
	r.POST("/addBlock", addBlockHandler)
	r.POST("/createShard", createShardHandler)

	r.POST("/addTransaction", addTransactionHandler)
	r.GET("/conflicts", getConflicts)
	r.DELETE("/removeLastBlock", removeLastBlock)
	r.POST("/assignNodesToShard", assignNodesToShardHandler)
	// Start Server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Starting REST API server on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error starting REST API server: %v", err)
		}
	}()

	// Graceful Shutdown Handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down REST API server...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("❌ Error shutting down server: %v", err)
	}
	log.Println("✅ Server shutdown complete.")
}

// API to check transaction status
func getTransactionStatus(c *gin.Context) {
	transactionID := c.Param("transactionID")

	transactionMu.Lock()
	defer transactionMu.Unlock()

	status, exists := transactionStatus[transactionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": status})
}

// Assign multiple nodes to a specific shard
func assignNodesToShardHandler(c *gin.Context) {
	var reqBody struct {
		ShardID int   `json:"shard_id"`
		Nodes   []int `json:"nodes"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if reqBody.ShardID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Shard ID"})
		return
	}
	// Assign selected nodes to the specified shard
	for i := range Blockchain {
		for _, nodeID := range reqBody.Nodes {
			if Blockchain[i].Index == nodeID {
				Blockchain[i].ShardID = reqBody.ShardID
			}
		}
	}
	log.Printf("✅ Assigned nodes %v to Shard %d", reqBody.Nodes, reqBody.ShardID)
	c.JSON(http.StatusOK, gin.H{"message": "Nodes assigned to shard successfully"})
}

// Get entire blockchain
func getBlockchain(c *gin.Context) {
	c.JSON(http.StatusOK, Blockchain)
}

// Get blockchain data for a specific shard
func getShardBlockchain(c *gin.Context) {
	shardID := c.Query("shardID")
	if shardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing shard ID"})
		return
	}

	shardNum, err := strconv.Atoi(shardID)
	if err != nil || shardNum < 0 || shardNum >= NumShards {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shard ID"})
		return
	}

	c.JSON(http.StatusOK, getShardBlocks(shardNum))
}

// Add a new block to the blockchain
func addBlockHandler(c *gin.Context) {
	var reqBody struct {
		ContainerID string `json:"containerID"`
	}
	// Parse request body
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	// Validate ContainerID
	if reqBody.ContainerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing container ID"})
		return
	}
	// Lock blockchain to ensure thread safety
	BlockchainMu.Lock()

	// Check if a block with the same container ID already exists
	for _, block := range Blockchain {
		if block.ContainerID == reqBody.ContainerID {
			BlockchainMu.Unlock()
			log.Printf("⚠️ Block with ContainerID %s already exists", reqBody.ContainerID)
			c.JSON(http.StatusConflict, gin.H{"error": "Block already exists"})
			return
		}
	}
	addBlock(reqBody.ContainerID)
	totalBlocks := len(Blockchain)
	BlockchainMu.Unlock()

	// Log blockchain state after modification
	log.Printf("✅ Block added: ContainerID %s | Total Blocks: %d", reqBody.ContainerID, totalBlocks)
	log.Printf("📌 Current Blockchain State: %+v", Blockchain)

	// Return success response
	c.JSON(http.StatusCreated, gin.H{"message": "Block added successfully", "total_blocks": totalBlocks})
}

func addTransactionHandler(c *gin.Context) {
	var reqBody struct {
		SourceBlock int    `json:"source"`
		TargetBlock int    `json:"target"`
		Data        string `json:"data"`
	}

	// Parse and validate request
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	// Generate a unique Transaction ID
	transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())

	// Store transaction as pending before executing
	transactionMu.Lock()
	transactionStatus[transactionID] = "pending"
	transactionMu.Unlock()

	// Run transaction processing in a separate goroutine (non-blocking)
	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data)

	// Immediately return response to the user
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Transaction submitted for processing",
		"transactionID": transactionID,
		"status":        "pending",
	})
}

// processTransaction executes a transaction asynchronously
func processTransaction(transactionID string, sourceBlockIdx int, targetBlockIdx int, data string) {
	BlockchainMu.Lock()

	// Ensure source block exists
	var sourceBlock *Block
	for i := range Blockchain {
		if Blockchain[i].Index == sourceBlockIdx {
			sourceBlock = &Blockchain[i]
			break
		}
	}
	// Ensure target block exists
	var targetBlockExists bool
	for _, block := range Blockchain {
		if block.Index == targetBlockIdx {
			targetBlockExists = true
			break
		}
	}
	BlockchainMu.Unlock()

	// Handle missing blocks
	if sourceBlock == nil || !targetBlockExists {
		transactionMu.Lock()
		transactionStatus[transactionID] = "failed"
		transactionMu.Unlock()
		log.Printf("❌ Transaction %s failed: Source or Target block missing", transactionID)
		return
	}

	// Simulate some processing time (e.g., validation, consensus, etc.)
	time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)

	// Add transaction safely
	transaction := Transaction{
		ContainerID:   strconv.Itoa(targetBlockIdx),
		Timestamp:     time.Now().Format(time.RFC3339),
		TransactionID: transactionID,
		Version:       len(sourceBlock.Transactions) + 1,
		Data:          data,
	}

	BlockchainMu.Lock()
	sourceBlock.Transactions = append(sourceBlock.Transactions, transaction)
	BlockchainMu.Unlock()

	// Update transaction status
	transactionMu.Lock()
	transactionStatus[transactionID] = "completed"
	transactionMu.Unlock()

	log.Printf("✅ Transaction %s completed: Block %d -> Block %d | Data: %s",
		transactionID, sourceBlockIdx, targetBlockIdx, data)
}

func createShardHandler(c *gin.Context) {
	var reqBody struct {
		Nodes []int `json:"nodes"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	for _, nodeID := range reqBody.Nodes {
		for i := range Blockchain {
			if Blockchain[i].Index == nodeID {
				Blockchain[i].ShardID = len(Blockchain) // Assign new shard
			}
		}
	}

	log.Printf("✅ New shard created with nodes: %v", reqBody.Nodes)
	c.JSON(http.StatusOK, gin.H{"message": "Shard created successfully"})
}

func resetBlockchainHandler(c *gin.Context) {
	for i := range Blockchain {
		Blockchain[i].ShardID = 0 // Reset all nodes to one shard
	}

	log.Println("✅ Blockchain reset to single linear chain.")
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain reset successfully"})
}

// Fetch concurrency conflicts
func getConflicts(c *gin.Context) {
	conflictsMu.Lock()
	defer conflictsMu.Unlock()

	log.Printf("🔍 Fetching concurrency conflicts. Total: %d", len(concurrencyConflicts))
	c.JSON(http.StatusOK, gin.H{
		"total_conflicts": len(concurrencyConflicts),
		"conflicts":       concurrencyConflicts,
	})
}

// Remove the last block from the blockchain
func removeLastBlock(c *gin.Context) {
	if len(Blockchain) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blockchain is empty, no blocks to remove"})
		return
	}

	BlockchainMu.Lock()
	Blockchain = Blockchain[:len(Blockchain)-1]
	BlockchainMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "Last block removed successfully"})
}

// Check if Docker is available
func checkDockerConnection(cli *client.Client) bool {
	_, err := cli.Ping(context.Background())
	if err != nil {
		log.Printf("⚠️ Docker connection failed: %v", err)
		return false
	}
	log.Println("✅ Docker connection successful")
	return true
}

func main() {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ Error creating Docker client: %v", err)
	}
	defer cli.Close()

	// Check if Docker is running
	if !checkDockerConnection(cli) {
		log.Fatal("❌ Docker is not running. Please start Docker and try again.")
	}

	// Handle OS signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Process containers if `--process` flag is passed
	if process {
		log.Println("📦 Processing containers...")
		processContainers(cli)
	}

	// Start REST API server if `--server` flag is passed
	if server {
		go runAPIServer(cli) // Run API server in a separate goroutine
	}

	// Wait for termination signal
	<-quit
	log.Println("🛑 Shutting down gracefully...")

	log.Println("✅ Server stopped successfully.")
}
