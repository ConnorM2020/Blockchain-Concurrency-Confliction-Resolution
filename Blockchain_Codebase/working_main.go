package main

import (
	"blockchain/blockchain_test"
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
	transactionLogs     []TransactionLog
	transactionLogsMu   sync.Mutex
	port                string
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

// API handler to trigger blockchain operations
func executeTransaction(c *gin.Context) {
	var request struct {
		Option int `json:"option"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	startTime := time.Now() // Start measuring execution time

	var message string
	var sourceBlock, targetBlock int

	switch request.Option {
	case 1:
		log.Println("⚡ Running Sharded Transactions...")
		sourceBlock, targetBlock = 1, 10 // Example: Adjust based on logic
		blockchain_test.ProcessSharded(10, 4)
		message = "Sharded Transactions Executed"
	case 2:
		log.Println("📜 Running Non-Sharded Transactions...")
		sourceBlock, targetBlock = 2, 15 // Example: Adjust based on logic
		blockchain_test.ProcessNonSharded(10)
		message = "Non-Sharded Transactions Executed"
	case 3:
		log.Println("🚀 Running Blockchain Stress Test...")
		blockchain_test.RunStressTest()
		message = "Stress Test Executed"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option selected"})
		return
	}
	// Compute Execution Time
	executionTime := time.Since(startTime).Seconds()

	// Send JSON response with execution details
	c.JSON(http.StatusOK, gin.H{
		"message":        message,
		"execution_time": executionTime,
		"source_block":   sourceBlock,
		"target_block":   targetBlock,
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

func processTransaction(transactionID string, source int, target int, data string, expectedSharded bool) {
	startTime := time.Now()

	// Verify actual shard assignments
	sourceShard := getShardID(fmt.Sprintf("%d", source))
	targetShard := getShardID(fmt.Sprintf("%d", target))
	isSharded := sourceShard != targetShard // Correct dynamic check

	// Log correct classification
	log.Printf("🔍 Processing Transaction: %s | Source: %d (Shard %d) → Target: %d (Shard %d) | Actual Sharded: %v",
		transactionID, source, sourceShard, target, targetShard, isSharded)

	// If a transaction was expected to be sharded but isn't, adjust the target
	if expectedSharded && !isSharded {
		log.Printf("⚠️ Mismatch! Expected sharded but assigned same shard. Adjusting target...")
		target = (target + 1) % 10 // Adjust target to ensure different shard
		targetShard = getShardID(fmt.Sprintf("%d", target))
		isSharded = sourceShard != targetShard
	}

	// Simulate processing time
	if isSharded {
		time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
	} else {
		time.Sleep(time.Duration(3+rand.Intn(4)) * time.Second)
	}

	BlockchainMu.Lock()
	defer BlockchainMu.Unlock()

	// Find the source block
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
	// Segmenting transaction data
	totalSegments := (len(data) + maxSegmentSize - 1) / maxSegmentSize
	TransactionMu.Lock()
	transactionSegments[transactionID] = make([]TransactionSegment, 0, totalSegments)
	TransactionMu.Unlock()

	// Process transaction segments in parallel
	var wg sync.WaitGroup
	for i := 0; i < totalSegments; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := i * maxSegmentSize
			end := start + maxSegmentSize
			if end > len(data) {
				end = len(data)
			}

			segment := TransactionSegment{
				TransactionID: transactionID,
				ShardID:       sourceShard,
				SegmentIndex:  i,
				TotalSegments: totalSegments,
				Data:          data[start:end],
			}

			TransactionMu.Lock()
			transactionSegments[transactionID] = append(transactionSegments[transactionID], segment)
			TransactionMu.Unlock()

			log.Printf("🔄 Processed segment %d/%d for transaction %s", i+1, totalSegments, transactionID)
		}(i)
	}
	wg.Wait()

	executionTime := time.Since(startTime).Seconds() * 1000 // Convert to ms

	// Finalizing the transaction
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
			Type: func() string {
				if isSharded {
					return "Sharded"
				}
				return "Non-Sharded"
			}(),
			ExecTime:  executionTime,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		sourceBlock.Transactions = append(sourceBlock.Transactions, newTransaction)
		transactionStatus[transactionID] = "completed"
		delete(transactionSegments, transactionID)

		log.Printf("✅ Transaction %s completed: Block %d → Block %d (Type: %s | Exec Time: %.3f ms)",
			transactionID, source, target, newTransaction.Type, executionTime)
	} else {
		transactionStatus[transactionID] = "in-progress"
	}
	TransactionMu.Unlock()
	// Logging the transaction
	transactionLogsMu.Lock()
	transactionType := map[bool]string{true: "Sharded", false: "Non-Sharded"}[isSharded]
	log.Printf("📝 Adding to Transaction Logs: ID=%s, Type=%s", transactionID, transactionType)

	transactionLogs = append(transactionLogs, TransactionLog{
		TxID:      transactionID,
		Source:    source,
		Target:    target,
		Type:      transactionType,
		ExecTime:  executionTime,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	transactionLogsMu.Unlock()
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

	log.Printf("📜 Fetching transaction logs: %d entries", len(transactionLogs))
	for _, logEntry := range transactionLogs {
		log.Printf("📝 Log Entry: %+v", logEntry)
	}

	c.JSON(http.StatusOK, gin.H{"logs": transactionLogs})
}

// Start the REST API server
func runAPIServer(cli *client.Client) {
	r := gin.Default()
	r.GET("/", listRoutes)

	r.Use(CORSMiddleware()) // Enable CORS
	// API Endpoints
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

	isSharded := getShardID(fmt.Sprintf("%d", reqBody.SourceBlock)) != getShardID(fmt.Sprintf("%d", reqBody.TargetBlock))

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

	isSharded := getShardID(fmt.Sprintf("%d", reqBody.SourceBlock)) != getShardID(fmt.Sprintf("%d", reqBody.TargetBlock))

	// Process transaction asynchronously
	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data, isSharded)
	fmt.Println("🔍 Transaction being added -> Source:", reqBody.SourceBlock, "Target:", reqBody.TargetBlock, "Sharded:", isSharded)

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
		isSharded := getShardID(fmt.Sprintf("%d", tx.Source)) != getShardID(fmt.Sprintf("%d", tx.Target))
		// Process transaction asynchronously
		go processTransaction(transactionID, tx.Source, tx.Target, tx.Data, isSharded)
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
		isSharded := getShardID(fmt.Sprintf("%d", tx.Source)) != getShardID(fmt.Sprintf("%d", tx.Target))

		go processTransaction(transactionID, tx.Source, tx.Target, tx.Data, isSharded)
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
		txCopy := tx // Avoid race conditions

		wg.Add(1)
		go func(txCopy ShardedTransaction) { // ✅ Use correct type
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

					// ✅ Determine if this is a sharded transaction
					isSharded := getShardID(fmt.Sprintf("%d", src)) != getShardID(fmt.Sprintf("%d", tgt))

					// ✅ Safely store transaction ID
					mu.Lock()
					transactionIDs = append(transactionIDs, transactionID)
					mu.Unlock()

					// ✅ Process transaction asynchronously
					go processTransaction(transactionID, src, tgt, txCopy.Data, isSharded)
				}
			}
		}(txCopy) // ✅ Correctly passes copy
	}

	wg.Wait() // ✅ Ensure all goroutines finish before responding

	// ✅ Respond with transaction IDs
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

	// Run Interactive CLI Execution in a separate goroutine
	go func() {
		for {
			fmt.Println("\n📌 Blockchain Execution Options:")
			fmt.Println("[1] Run Sharded Transactions")
			fmt.Println("[2] Run Non-Sharded Transactions")
			fmt.Println("[3] Run Stress Test")
			fmt.Println("[4] Exit")
			fmt.Print("> ")

			var choice int
			_, err := fmt.Scanln(&choice)
			if err != nil {
				fmt.Println("⚠️ Invalid input. Please enter a number (1-4).")
				continue
			}

			switch choice {
			case 1:
				fmt.Println("⚡ Running Sharded Transactions...")

				blockchain_test.ProcessSharded(10, 4)
			case 2:
				fmt.Println("📜 Running Non-Sharded Transactions...")
				blockchain_test.ProcessNonSharded(10)
			case 3:
				fmt.Println("🚀 Running Blockchain Stress Test...")
				blockchain_test.RunStressTest()
			case 4:
				fmt.Println("Exiting interactive mode...")
				return
			default:
				fmt.Println("⚠️ Invalid choice. Please select a valid option.")
			}
		}
	}()

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
