package main

import (
	//"blockchain/blockchain_test"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
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
	TxID        string  `json:"txID"`
	Source      int     `json:"source"`
	Target      int     `json:"target"`
	Message     string  `json:"message"`
	Type        string  `json:"type"`
	ExecTime    float64 `json:"execTime"`
	Finality    float64 `json:"finalityTime"`
	Timestamp   string  `json:"timestamp"`
	Propagation float64 `json:"propagationLatency"`
	TPS         float64 `json:"tps"`
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

	// Performance measurement benchmarks
	txCount      int
	txCountMutex sync.Mutex
	currentTPS   float64
)

var executionOptions = []string{"Run Sharded Transactions", "Run Non-Sharded Transactions", "Run Stress Test"}

// Monitors and calculates transactions per second (TPS)
func monitorTPS() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		txCountMutex.Lock()
		currentTPS = float64(txCount)
		txCount = 0
		txCountMutex.Unlock()
	}
}

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

	var (
		message     string
		isSharded   bool
		sourceBlock int
		targetBlock int
	)

	sourceBlock = rand.Intn(10) + 1

	switch request.Option {
	case 1: // Sharded
		log.Println("⚡ Running Sharded Transactions...")
		isSharded = true
		message = "Sharded Transactions Executed"

		// Choose a target in a DIFFERENT shard and NOT equal to source
		for {
			targetBlock = rand.Intn(10) + 1
			if targetBlock != sourceBlock &&
				getShardID(fmt.Sprintf("%d", sourceBlock)) != getShardID(fmt.Sprintf("%d", targetBlock)) {
				break
			}
		}

	case 2: // Non-Sharded
		log.Println("📜 Running Non-Sharded Transactions...")
		isSharded = false
		message = "Non-Sharded Transactions Executed"
		sourceShard := getShardID(fmt.Sprintf("%d", sourceBlock))

		// Find a target block that is in the same shard and not the same as source
		for {
			targetBlock = rand.Intn(10) + 1
			targetShard := getShardID(fmt.Sprintf("%d", targetBlock))
			if targetBlock != sourceBlock && targetShard == sourceShard {
				break
			}
		}

		// Optional log for clarity
		log.Printf("✅ Non-Sharded Tx: Source Block %d [Shard %d] → Target Block %d [Shard %d]",
			sourceBlock, sourceShard, targetBlock, getShardID(fmt.Sprintf("%d", targetBlock)))

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option selected"})
		return
	}

	transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	go processTransaction(transactionID, sourceBlock, targetBlock, "Transaction Data", isSharded)

	executionTime := time.Since(startTime).Seconds()
	finalityTime := executionTime * 1000 // in ms

	log.Printf("Finality time for %s: %.2f ms", transactionID, finalityTime)

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

		executionTime := time.Since(startTime).Seconds() * 1000 // ms

		// Simulate propagation delay based on shard distance
		var propagationLatency float64
		if isSharded {
			propagationLatency = float64(10 + rand.Intn(15)) // Sharded: 10–25ms
		} else {
			propagationLatency = float64(40 + rand.Intn(30)) // Non-sharded: 40–70ms
		}
		

		// Simulate consensus delay (e.g., 2–4 validators * 30ms)
		consensusDelay := float64((2 + rand.Intn(3)) * 30) // 60–120 ms

		// Finality = Execution + Consensus + Propagation
		finalityTime := executionTime + consensusDelay + propagationLatency

		log.Printf("🕒 Finality time for %s: %.2f ms (Exec: %.2f + Consensus: %.2f + Propagation: %.2f)",
			transactionID, finalityTime, executionTime, consensusDelay, propagationLatency)


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
			Propagation:   propagationLatency,
			//TPS: 			tps,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		transactionStatus[transactionID] = "completed"
		TransactionMu.Unlock()

		tps := 1000.0 / executionTime
		tps = math.Round(tps*100) / 100 // Optional rounding

		// save to firebase context
		SaveTransactionToFirestore(TransactionLog{
			TxID:        transactionID,
			Source:      source,
			Target:      target,
			Message:     data,
			Type:        typeLabel,
			ExecTime:    executionTime,
			Finality:    finalityTime,
			Propagation: propagationLatency,
			Timestamp:   time.Now().Format(time.RFC3339),
			TPS:         tps,
		})

		transactionLogsMu.Lock()
		// Log to global transaction history
		transactionLogs = append(transactionLogs, TransactionLog{
			TxID:        transactionID,
			Source:      source,
			Target:      target,
			Message:     data,
			Type:        typeLabel,
			ExecTime:    executionTime,
			Finality:    finalityTime,
			Propagation: propagationLatency,
			Timestamp:   time.Now().Format(time.RFC3339),
			TPS:         tps, // ✅ This was missing
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
	visualiseShards()
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
	r.POST("/deadlocksim", simulateDeadlockHandler)

	r.POST("/addBlock", addBlockHandler)
	r.POST("/createShard", createShardHandler)
	r.POST("/addTransactionSegment", addTransactionSegmentHandler)
	r.POST("/addTransaction", addTransactionHandler)               // Ensure this calls the correct handler
	r.POST("/addShardedTransaction", addShardedTransactionHandler) // Use different endpoint
	r.POST("/addParallelTransactions", addParallelTransactionsHandler)
	r.POST("/assignNodesToShard", assignNodesToShardHandler)
	r.POST("/shardTransactions", shardTransactionsHandler)
	r.DELETE("/removeLastBlock", removeLastBlock)

	// Performance measurements
	r.GET("/metrics/tps", func(c *gin.Context) {
		txCountMutex.Lock()
		defer txCountMutex.Unlock()
		c.JSON(http.StatusOK, gin.H{"tps": currentTPS})
	})

	// Start Server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Starting REST API server on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting REST API server: %v", err)
		}
	}()

	// Graceful Shutdown Handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down REST API server...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server shutdown complete.")
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
			"/deadlocksim",
			"/addParallelTransactions",
			"/assignNodesToShard",
			"/shardTransactions",
			"/removeLastBlock",
			"/getTransactionStatus",
		},
	})
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
	if reqBody.SourceBlock == reqBody.TargetBlock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source and target block cannot be the same"})
		return
	}

	transactionID := fmt.Sprintf("tx-%d", time.Now().UnixNano())

	// Store transaction as pending
	TransactionMu.Lock()
	transactionStatus[transactionID] = "pending"
	TransactionMu.Unlock()

	isSharded := reqBody.Type == "sharded"

	// Process transaction asynchronously
	go processTransaction(transactionID, reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data, isSharded)

	log.Printf("Sharded Transaction being added -> Source: %d | Target: %d | Data: %s", reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data)

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
	// ✅ Prevent node from sending to itself
	if reqBody.SourceBlock == reqBody.TargetBlock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source and target block cannot be the same"})
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
			if tx.Source == tx.Target {
				log.Printf("❌ Skipping self-node transaction: %d → %d", tx.Source, tx.Target)
				return
			}

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
					if src == tgt {
						log.Printf("❌ Skipping self-node sharded transaction: %d → %d", src, tgt)
						continue
					}

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
		Nodes []int `json:"nodes"` // selected block indices
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if len(reqBody.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No nodes provided"})
		return
	}

	// Step 1: Find the highest existing shard ID
	highestShard := 0
	for _, block := range Blockchain {
		if block.ShardID > highestShard {
			highestShard = block.ShardID
		}
	}
	newShardID := highestShard + 1

	// Step 2: Assign selected nodes to the new shard
	for i := range Blockchain {
		for _, nodeID := range reqBody.Nodes {
			if Blockchain[i].Index == nodeID {
				Blockchain[i].ShardID = newShardID
			}
		}
	}

	log.Printf("✅ Assigned nodes %v to NEW Shard %d", reqBody.Nodes, newShardID)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Nodes assigned to new shard successfully",
		"shard_id": newShardID,
		"node_ids": reqBody.Nodes,
	})
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
	log.Println("Docker connection successful")
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

// Simulate a deadlock with two non-sharded transactions (dynamically chosen)
func simulateDeadlockHandler(c *gin.Context) {
	var req struct {
		Source int `json:"source"`
		Target int `json:"target"`
	}

	// Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Expected JSON body with 'source' and 'target' integers."})
		return
	}

	if req.Source == req.Target {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source and target must be different blocks."})
		return
	}

	var blockA, blockB *Block

	// Find selected blocks from Blockchain
	BlockchainMu.Lock()
	for i := range Blockchain {
		if Blockchain[i].Index == req.Source {
			blockA = &Blockchain[i]
		} else if Blockchain[i].Index == req.Target {
			blockB = &Blockchain[i]
		}
	}
	BlockchainMu.Unlock()

	if blockA == nil || blockB == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not find both source and target blocks in the blockchain."})
		return
	}

	// Create two interlocked pending transactions to simulate deadlock
	startTime1 := time.Now()
	startTime2 := startTime1.Add(2 * time.Second)

	tx1 := Transaction{
		TransactionID: fmt.Sprintf("deadlock-tx-%d", time.Now().UnixNano()),
		Source:        blockA.Index,
		Target:        blockB.Index,
		Data:          "Deadlock Tx A→B",
		Type:          "Non-Sharded",
		Status:        "pending", // Never completed
		Timestamp:     startTime1.Format(time.RFC3339),
	}
	tx2 := Transaction{
		TransactionID: fmt.Sprintf("deadlock-tx-%d", time.Now().UnixNano()+1),
		Source:        blockB.Index,
		Target:        blockA.Index,
		Data:          "Deadlock Tx B→A",
		Type:          "Non-Sharded",
		Status:        "pending",
		Timestamp:     startTime2.Format(time.RFC3339),
	}

	// Store in global logs/status without marking completed
	TransactionMu.Lock()
	transactionStatus[tx1.TransactionID] = "pending"
	transactionStatus[tx2.TransactionID] = "pending"

	transactionLogs = append(transactionLogs,
		TransactionLog{
			TxID:      tx1.TransactionID,
			Source:    tx1.Source,
			Target:    tx1.Target,
			Type:      "deadlock",
			ExecTime:  0,
			Timestamp: tx1.Timestamp,
		},
		TransactionLog{
			TxID:      tx2.TransactionID,
			Source:    tx2.Source,
			Target:    tx2.Target,
			Type:      "deadlock",
			ExecTime:  0,
			Timestamp: tx2.Timestamp,
		},
	)

	TransactionMu.Unlock()

	// Add pending transactions to respective blocks
	BlockchainMu.Lock()
	blockA.Transactions = append(blockA.Transactions, tx1)
	blockB.Transactions = append(blockB.Transactions, tx2)
	BlockchainMu.Unlock()

	log.Printf("Deadlock simulation started between Block %d and Block %d", blockA.Index, blockB.Index)

	c.JSON(http.StatusOK, gin.H{
		"message": "Deadlock simulation started between selected nodes.",
		"tx1":     tx1,
		"tx2":     tx2,
	})
}

// Main function
func main() {
	InitFirebase()                                    // initalise the firebase permanent storage
	transactionLogs = LoadTransactionsFromFirestore() // load existing system

	// Start TPS monitoring in the background
	go monitorTPS()
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
	defer firestoreClient.Close() // gracefully close with app
	log.Println("✅ Server shutdown complete.")
}
