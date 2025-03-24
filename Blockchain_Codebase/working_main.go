package main

import (
	//"blockchain/blockchain_test"
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

// Define TransactionLog structure
type TransactionLog struct {
	TxID      string  `json:"txID"`
	Source    int     `json:"source"`
	Target    int     `json:"target"`
	Type      string  `json:"type"`
	ExecTime  float64 `json:"execTime"`
	Timestamp string  `json:"timestamp"`
}

const (
	blockchainFile  = "blockchain.json"
	maxRetries      = 3
	maxSegmentSize  = 10
	shutdownTimeout = 5 * time.Second
)

var (
	transactionLogs   []TransactionLog
	transactionLogsMu sync.Mutex
	port              string

	process             bool
	server              bool
	TransactionPool     = make(map[string]*Transaction)
	transactionStatus   = make(map[string]string)
	TransactionMu       sync.Mutex
	transactionSegments = make(map[string][]TransactionSegment)
)

var executionOptions = []string{"Run Sharded Transactions", "Run Non-Sharded Transactions", "Run Stress Test"}

// return API handler
func getExecutionOptions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"options": executionOptions})
}

func executeTransaction(c *gin.Context) {
	var request struct {
		Option int `json:"option"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	startTime := time.Now()

	var message string
	var isSharded bool
	var sourceBlock, targetBlock int

	sourceBlock = rand.Intn(10) + 1
	targetBlock = rand.Intn(10) + 1
	for targetBlock == sourceBlock {
		targetBlock = rand.Intn(10) + 1
	}

	switch request.Option {
	case 1:
		log.Println("⚡ Running Sharded Transactions...")
		isSharded = true
		message = "Sharded Transactions Executed"

		for getShardID(fmt.Sprintf("%d", sourceBlock)) == getShardID(fmt.Sprintf("%d", targetBlock)) {
			targetBlock = rand.Intn(10) + 1
		}

	case 2:
		log.Println("📜 Running Non-Sharded Transactions...")
		isSharded = false
		message = "Non-Sharded Transactions Executed"

		sourceShard := getShardID(fmt.Sprintf("%d", sourceBlock))
		for getShardID(fmt.Sprintf("%d", targetBlock)) != sourceShard {
			targetBlock = rand.Intn(10) + 1
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option selected"})
		return
	}

	transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	go processTransaction(transactionID, sourceBlock, targetBlock, "Transaction Data", isSharded)

	executionTime := time.Since(startTime).Seconds()

	c.JSON(http.StatusOK, gin.H{
		"message":        message,
		"execution_time": executionTime,
		"source_block":   sourceBlock,
		"target_block":   targetBlock,
		"is_sharded":     isSharded,
	})
}



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

func processTransaction(transactionID string, source int, target int, data string, isSharded bool) {
	startTime := time.Now()

	//  Enforce correct sharding logic
	if isSharded {
		// Force different shards
		for getShardID(fmt.Sprintf("%d", target)) == getShardID(fmt.Sprintf("%d", source)) {
			target = rand.Intn(10) + 1
		}
	} else {
		// Force same shard
		for getShardID(fmt.Sprintf("%d", target)) != getShardID(fmt.Sprintf("%d", source)) {
			target = rand.Intn(10) + 1
		}
	}

	// Use the intended sharding type for labelling
	typeLabel := map[bool]string{true: "Sharded", false: "Non-Sharded"}[isSharded]

	// Simulate execution in a goroutine
	go func() {
		// Simulate processing time
		if isSharded {
			time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
		} else {
			time.Sleep(time.Duration(3+rand.Intn(4)) * time.Second)
		}

		BlockchainMu.Lock()
		defer BlockchainMu.Unlock()

		var sourceBlock *Block
		for i := range Blockchain {
			if Blockchain[i].Index == source {
				sourceBlock = &Blockchain[i]
				break
			}
		}

		if sourceBlock == nil {
			log.Printf("❌ ERROR: Source block %d not found for transaction %s", source, transactionID)
			TransactionMu.Lock()
			transactionStatus[transactionID] = "failed"
			TransactionMu.Unlock()
			return
		}

		executionTime := time.Since(startTime).Seconds() * 1000 // Convert to ms

		// Update block with transaction
		TransactionMu.Lock()
		sourceBlock.Transactions = append(sourceBlock.Transactions, Transaction{
			TransactionID: transactionID,
			Source:        source,
			Target:        target,
			Data:          data,
			Status:        "completed",
			Type:          typeLabel,
			ExecTime:      executionTime,
			Timestamp:     time.Now().Format(time.RFC3339),
		})
		transactionStatus[transactionID] = "completed"
		TransactionMu.Unlock()

		// Log to global transaction history
		transactionLogsMu.Lock()
		transactionLogs = append(transactionLogs, TransactionLog{
			TxID:      transactionID,
			Source:    source,
			Target:    target,
			Type:      typeLabel,
			ExecTime:  executionTime,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		transactionLogsMu.Unlock()

		log.Printf("✅ Transaction %s completed: Block %d → Block %d (Type: %s | Exec Time: %.3f ms)",
			transactionID, source, target, typeLabel, executionTime)
	}()
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

// API to fetch transaction logs
func getTransactionLogs(c *gin.Context) {
	transactionLogsMu.Lock()
	defer transactionLogsMu.Unlock()

	if transactionLogs == nil {
		transactionLogs = []TransactionLog{}
	}

	// Log transaction count & contents
	log.Printf("📜 Fetching transaction logs: %d entries", len(transactionLogs))
	for _, logEntry := range transactionLogs {
		log.Printf("📝 Log Entry: %+v", logEntry)
	}

	//  Send JSON response
	c.JSON(http.StatusOK, gin.H{"logs": transactionLogs})
}

// Start the REST API server
func runAPIServer(cli *client.Client) {
	r := gin.Default()
	r.GET("/", listRoutes)

	r.Use(CORSMiddleware()) // Enable CORS
	// API Endpoints
	r.GET("/allTransactions", getTransactionLogs)
	r.GET("/blockchain", getBlockchain)
	r.GET("/blockchain/shard", getShardBlockchain)
	r.GET("/transactionStatus/:transactionID", checkTransactionStatus)
	r.GET("/conflicts", getConflicts)
	r.GET("/transactionLogs", getTransactionLogs)

	r.GET("/executionOptions", getExecutionOptions)
	r.POST("/executeTransaction", executeTransaction)

	r.POST("/resetBlockchain", resetBlockchainHandler)
	r.POST("/addBlock", addBlockHandler)
	r.POST("/createShard", createShardHandler)
	r.POST("/addTransactionSegment", addTransactionSegmentHandler)

	r.POST("/addTransaction", addTransactionHandler)               // Ensure this calls the correct handler
	r.POST("/addShardedTransaction", addShardedTransactionHandler) // Use different endpoint

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

// Root endpoint to list all available routes
func listRoutes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"endpoints": []string{
			"/executionOptions",
			"/executeTransaction",
			"/blockchain",
			"/blockchain/shard",
			"/transactionStatus/:transactionID",
			"/conflicts",
			"/transactionLogs",
			"/resetBlockchain",
			"/addShardedTransactionHandler",
			"/addBlock",
			"/createShard",
			"/addTransactionSegment",
			"/addTransaction",
			"/addParallelTransactions",
			"/assignNodesToShard",
			"/shardTransactions",
			"/removeLastBlock",
			"/getTransactionStatus",
		},
	})
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

func addShardedTransactionHandler(c *gin.Context) {
	var reqBody struct {
		SourceBlock int    `json:"source"`
		TargetBlock int    `json:"target"`
		Data        string `json:"data"`
		Type        string `json:"type"`
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

	isSharded := reqBody.Type == "sharded"

	// Process transaction asynchronously
	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data, isSharded)

	log.Printf("🔍 Sharded Transaction being added -> Source: %d | Target: %d | Data: %s", reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data)

	// Return response immediately
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Sharded transaction submitted for processing",
		"transactionID": transactionID,
		"status":        "pending",
	})
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
		IsSharded   bool   `json:"is_sharded"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())

	TransactionMu.Lock()
	transactionStatus[transactionID] = "pending"
	TransactionMu.Unlock()

	isSharded := reqBody.IsSharded // ✅ Use the frontend’s instruction

	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data, isSharded)

	log.Printf("Transaction being added -> Source: %d | Target: %d | Sharded: %v", reqBody.SourceBlock, reqBody.TargetBlock, isSharded)

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Transaction submitted for processing",
		"transactionID": transactionID,
		"status":        "pending",
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
	var wg sync.WaitGroup
	transactionIDs := make([]string, 0)
	mu := sync.Mutex{}
	for _, tx := range transactions {
		wg.Add(1)
		go func(tx Transaction) {
			defer wg.Done()

			transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
			TransactionMu.Lock()
			transactionStatus[transactionID] = "pending"
			TransactionMu.Unlock()

			isSharded := getShardID(fmt.Sprintf("%d", tx.Source)) != getShardID(fmt.Sprintf("%d", tx.Target))

			mu.Lock()
			transactionIDs = append(transactionIDs, transactionID)
			mu.Unlock()
			// Process in a separate Goroutine
			go processTransaction(transactionID, tx.Source, tx.Target, tx.Data, isSharded)

		}(tx)
	}
	wg.Wait()
	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Parallel transactions are being processed",
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
		txCopy := tx // Avoid race conditions

		wg.Add(1)
		go func(txCopy ShardedTransaction) {
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

					// Ensure it's always sharded
					isSharded := true
					//isSharded := getShardID(fmt.Sprintf("%d", src)) != getShardID(fmt.Sprintf("%d", tgt))

					// Store transaction ID safely
					mu.Lock()
					transactionIDs = append(transactionIDs, transactionID)
					mu.Unlock()

					// Process transaction asynchronously
					go processTransaction(transactionID, src, tgt, txCopy.Data, isSharded)
				}
			}
		}(txCopy) // Correctly passes copy
	}

	wg.Wait() // Ensure all goroutines finish before responding

	// Return response
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

// Shutdown function to gracefully stop the API server
func shutdownAPIServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	srv := &http.Server{
		Addr: ":" + port,
	}

	return srv.Shutdown(ctx)
}

// Main function
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

	// Start the REST API Server so React frontend can connect
	log.Println("🚀 Starting Blockchain REST API on localhost:8080...")
	go runAPIServer(cli) // Start the API server in a separate goroutine

	// Wait a moment to ensure the API is up before processing containers
	select {
	case <-quit:
		log.Println("🛑 Received shutdown signal before starting execution.")
		return
	default:
	}

	// Check if `--process` flag is passed and process containers
	if process {
		log.Println("📦 Processing containers...")
		processContainers(cli)
	}

	// Wait for termination signal
	<-quit
	log.Println("🛑 Shutting down gracefully...")

	// Graceful shutdown of API server
	log.Println("🛑 Shutting down REST API server...")
	if err := shutdownAPIServer(); err != nil {
		log.Fatalf("❌ Error shutting down server: %v", err)
	}
	log.Println("✅ Server shutdown complete.")
}
