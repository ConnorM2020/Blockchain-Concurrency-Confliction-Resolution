package main

import (
	"context"
	"flag"
	"fmt"
	"log"
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

// Start the REST API server
func runAPIServer(cli *client.Client) {
	r := gin.Default()
	r.Use(CORSMiddleware()) // Enable CORS

	// API Endpoints
	r.GET("/blockchain", getBlockchain)
	r.GET("/blockchain/shard", getShardBlockchain)
	r.POST("/addBlock", addBlockHandler)
	r.POST("/addTransaction", addTransactionHandler)
	r.GET("/conflicts", getConflicts)
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
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if reqBody.ContainerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing container ID"})
		return
	}
	addBlock(reqBody.ContainerID)
	c.JSON(http.StatusCreated, gin.H{"message": "Block added successfully"})
}

func addTransactionHandler(c *gin.Context) {
	var reqBody struct {
		SourceBlock int    `json:"source"`
		TargetBlock int    `json:"target"`
		Data        string `json:"data"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	// Ensure source block exists
	var sourceBlock *Block
	for i := range Blockchain {
		if Blockchain[i].Index == reqBody.SourceBlock {
			sourceBlock = &Blockchain[i]
			break
		}
	}
	if sourceBlock == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source block not found"})
		return
	}
	// Ensure target block exists
	var targetBlockExists bool
	for _, block := range Blockchain {
		if block.Index == reqBody.TargetBlock {
			targetBlockExists = true
			break
		}
	}
	if !targetBlockExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target block not found"})
		return
	}

	// Add transaction to the source block
	transaction := Transaction{
		ContainerID:   strconv.Itoa(reqBody.TargetBlock), // Using TargetBlock ID
		Timestamp:     time.Now().Format(time.RFC3339),
		TransactionID: fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		Version:       len(sourceBlock.Transactions) + 1, // Ensuring versioning
	}

	sourceBlock.Transactions = append(sourceBlock.Transactions, transaction)

	log.Printf("✅ Transaction added: Block %d -> Block %d | Data: %s",
		reqBody.SourceBlock, reqBody.TargetBlock, reqBody.Data)

	c.JSON(http.StatusCreated, gin.H{"message": "Transaction added successfully"})
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
