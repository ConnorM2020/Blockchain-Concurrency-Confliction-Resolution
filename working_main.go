package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	blockchainFile = "blockchain.json"

	maxRetries = 3
)

type Shard struct {
	mu sync.Mutex
}

var (
	shards  [numShards]Shard
	port    string
	process bool
	server  bool
)

func init() {
	// Define CLI flags
	flag.StringVar(&port, "port", "8080", "Port number to run the server")
	flag.BoolVar(&process, "process", false, "Process containers and add to blockchain")
	flag.BoolVar(&server, "server", false, "Run REST API server for inspecting containers")
	flag.Parse()
}

// Process running Docker containers and add them to the blockchain
func processContainers(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Error listing containers: %v", err)
	}

	fmt.Println("Running Docker Containers:")
	for _, container := range containers {
		fmt.Printf("Container ID: %s\nImage: %s\nState: %s\nStatus: %s\n",
			container.ID, container.Image, container.State, container.Status)
		fmt.Println("----------------------------------------------------")
	}

	var wg sync.WaitGroup
	action := func(containerID string) {
		log.Printf("Processing container: %s", containerID)
		addBlock(containerID)

		containerInfo, err := cli.ContainerInspect(context.Background(), containerID)
		if err != nil {
			log.Printf("Error inspecting container %s: %v", containerID, err)
			return
		}
		log.Printf("Container %s status: %s", containerID, containerInfo.State.Status)
	}

	for _, container := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			action(containerID)
		}(container.ID)
	}

	wg.Wait()
	log.Println("Finished processing all containers.")
	displayBlockchain()
}

// Start the REST API server
func runAPIServer(cli *client.Client) {
	mux := http.NewServeMux()

	// Fetch Blockchain
	mux.HandleFunc("/blockchain", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(blockchain)
	})

	// Add a Block
	mux.HandleFunc("/addBlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
			return
		}

		var reqBody struct {
			ContainerID string `json:"containerID"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		if reqBody.ContainerID == "" {
			http.Error(w, `{"error": "Missing container ID"}`, http.StatusBadRequest)
			return
		}

		addBlock(reqBody.ContainerID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Block added successfully"})
	})
	// Add a Transaction
	mux.HandleFunc("/addTransaction", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
			return
		}

		var reqBody struct {
			ContainerID string `json:"containerID"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		if reqBody.ContainerID == "" {
			http.Error(w, `{"error": "Missing container ID"}`, http.StatusBadRequest)
			return
		}
		if err := addTransaction(reqBody.ContainerID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err), http.StatusConflict)
			return
		}

		// Add transaction separately
		addTransaction(reqBody.ContainerID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Transaction added successfully"})
	})

	mux.HandleFunc("/conflicts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
			return
		}
		conflictsMu.Lock()
		defer conflictsMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_conflicts": len(concurrencyConflicts),
			"conflicts":       concurrencyConflicts,
		})
	})

	// Remove Last Block
	mux.HandleFunc("/removeLastBlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
			return
		}

		if len(blockchain) == 0 {
			http.Error(w, `{"error": "Blockchain is empty, no blocks to remove"}`, http.StatusBadRequest)
			return
		}

		blockchainMu.Lock()
		blockchain = blockchain[:len(blockchain)-1]
		blockchainMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Last block removed successfully"})
	})

	server := &http.Server{Addr: ":" + port, Handler: mux}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting REST API server on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting REST API server: %v", err)
		}
	}()

	// Graceful Shutdown Handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down REST API server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server shutdown complete.")
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	// Process containers if `--process` flag is passed
	if process {
		processContainers(cli)
	}

	// Start REST API server if `--server` flag is passed
	if server {
		runAPIServer(cli)
	}
}
