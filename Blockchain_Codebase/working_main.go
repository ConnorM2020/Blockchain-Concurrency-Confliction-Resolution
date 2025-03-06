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
	maxSegmentSize = 10
)

var (
	port                string
	process             bool
	server              bool
	TransactionPool     = make(map[string]*Transaction)
	transactionStatus   = make(map[string]string)
	TransactionMu       sync.Mutex
	transactionSegments = make(map[string][]TransactionSegment)
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

// Process a transaction asynchronously with segmentation
func processTransaction(transactionID string, source int, target int, data string) {
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

	// Lock blockchain
	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	// Find source block
	var sourceBlock *Block
	for i := range Blockchain {
		if Blockchain[i].Index == source {
			sourceBlock = &Blockchain[i]
			break
		}
	}

	if sourceBlock == nil {
		log.Printf("⚠️ Source block %d not found for transaction %s", source, transactionID)
		TransactionMu.Lock()
		transactionStatus[transactionID] = "failed"
		TransactionMu.Unlock()
		return
	}

	// Split large transactions into segments
	totalSegments := (len(data) + maxSegmentSize - 1) / maxSegmentSize

	TransactionMu.Lock()
	transactionSegments[transactionID] = make([]TransactionSegment, 0, totalSegments)
	TransactionMu.Unlock()

	for i := 0; i < totalSegments; i++ {
		start := i * maxSegmentSize
		end := start + maxSegmentSize
		if end > len(data) {
			end = len(data)
		}

		segment := TransactionSegment{
			TransactionID: transactionID,
			ShardID:       getShardID(fmt.Sprintf("%d", source)),
			SegmentIndex:  i,
			TotalSegments: totalSegments,
			Data:          data[start:end],
		}

		TransactionMu.Lock()
		transactionSegments[transactionID] = append(transactionSegments[transactionID], segment)
		TransactionMu.Unlock()

		log.Printf("🔄 Processed segment %d/%d for transaction %s", i+1, totalSegments, transactionID)
	}

	// Reconstruct transaction once all segments arrive
	TransactionMu.Lock()
	if len(transactionSegments[transactionID]) == totalSegments {
		fullData := ""
		for _, seg := range transactionSegments[transactionID] {
			fullData += seg.Data
		}
		newTransaction := Transaction{
			TransactionID: transactionID,
			Source:        source,
			Target:        target,
			Data:          fullData,
			Status:        "completed",
		}

		sourceBlock.Transactions = append(sourceBlock.Transactions, newTransaction)
		transactionStatus[transactionID] = "completed"
		delete(transactionSegments, transactionID) // Cleanup segments

		log.Printf("✅ Transaction %s fully reconstructed and completed: Block %d → Block %d", transactionID, source, target)
	} else {
		transactionStatus[transactionID] = "in-progress"
	}
	TransactionMu.Unlock()
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

// Check transaction status
func checkTransactionStatus(c *gin.Context) {
	transactionID := c.Param("transactionID")

	TransactionMu.Lock()
	status, exists := transactionStatus[transactionID]
	TransactionMu.Unlock()

	if exists {
		c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": status})
	} else {
		c.JSON(http.StatusOK, gin.H{"transaction_id": transactionID, "status": "pending"})
	}
}

// Start the REST API server
func runAPIServer(cli *client.Client) {
	r := gin.Default()
	r.Use(CORSMiddleware()) // Enable CORS
	// API Endpoints
	r.GET("/blockchain", getBlockchain)
	r.GET("/blockchain/shard", getShardBlockchain)
	r.GET("/transactionStatus/:transactionID", checkTransactionStatus)
	r.GET("/conflicts", getConflicts)

	r.POST("/resetBlockchain", resetBlockchainHandler)
	r.POST("/addBlock", addBlockHandler)
	r.POST("/createShard", createShardHandler)
	r.POST("/addTransactionSegment", addTransactionSegmentHandler)
	r.POST("/addTransaction", addTransactionHandler)
	r.POST("/addParallelTransactions", addParallelTransactionsHandler)
	r.POST("/assignNodesToShard", assignNodesToShardHandler)
	r.POST("/shardTransactions", shardTransactionsHandler)

	r.DELETE("/removeLastBlock", removeLastBlock)

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

// Add a single transaction
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

	// Store transaction as pending
	TransactionMu.Lock()
	transactionStatus[transactionID] = "pending"
	TransactionMu.Unlock()

	// Process transaction asynchronously
	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data)

	// Return response immediately
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Transaction submitted for processing",
		"transactionID": transactionID,
		"status":        "pending",
	})
}

// Add multiple transactions in parallel
func addParallelTransactions(c *gin.Context) {
	var transactions []Transaction
	if err := c.ShouldBindJSON(&transactions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if len(transactions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No transactions provided"})
		return
	}
	var transactionIDs []string
	for _, tx := range transactions {
		transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
		tx.TransactionID = transactionID

		TransactionMu.Lock()
		transactionStatus[transactionID] = "pending"
		TransactionPool[transactionID] = &tx
		TransactionMu.Unlock()

		transactionIDs = append(transactionIDs, transactionID)

		// Process transaction asynchronously
		go processTransaction(transactionID, tx.Source, tx.Target, tx.Data)
	}
	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Transactions are being processed",
		"transactionIDs": transactionIDs,
	})
}

func addParallelTransactionsHandler(c *gin.Context) {
	var transactions []Transaction
	if err := c.ShouldBindJSON(&transactions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if len(transactions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No transactions provided"})
		return
	}

	var transactionIDs []string

	for _, tx := range transactions {
		transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
		tx.TransactionID = transactionID

		TransactionMu.Lock()
		transactionStatus[transactionID] = "pending"
		TransactionPool[transactionID] = &tx
		TransactionMu.Unlock()

		transactionIDs = append(transactionIDs, transactionID)

		// Process transaction asynchronously
		go processTransaction(transactionID, tx.Source, tx.Target, tx.Data)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Transactions are being processed",
		"transactionIDs": transactionIDs,
	})
}

func shardTransactionsHandler(c *gin.Context) {
	var req struct {
		Transactions []ShardedTransaction `json:"transactions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if len(req.Transactions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No transactions provided"})
		return
	}

	var wg sync.WaitGroup
	transactionIDs := make([]string, 0)
	mu := sync.Mutex{}

	// Iterate over the transactions
	for _, tx := range req.Transactions {
		txCopy := tx // ✅ Now txCopy is a proper copy of the defined struct

		wg.Add(1)
		go func(txCopy ShardedTransaction) { // ✅ Use the correct type in function argument
			defer wg.Done()

			for _, src := range txCopy.Source {
				for _, tgt := range txCopy.Target {
					transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())

					TransactionMu.Lock()
					TransactionPool[transactionID] = &Transaction{
						Source:        src,
						Target:        tgt,
						Data:          txCopy.Data,
						TransactionID: transactionID,
						Status:        "pending",
					}
					TransactionMu.Unlock()

					// Safely store transaction ID
					mu.Lock()
					transactionIDs = append(transactionIDs, transactionID)
					mu.Unlock()

					// Process transaction asynchronously
					go processTransaction(transactionID, src, tgt, txCopy.Data)
				}
			}
		}(txCopy) // ✅ Now it properly passes the copy
	}

	wg.Wait()

	// Respond with transaction IDs
	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Sharded transactions are being processed",
		"transactionIDs": transactionIDs,
	})
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
