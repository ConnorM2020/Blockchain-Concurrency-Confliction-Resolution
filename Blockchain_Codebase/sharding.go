package main

import (
	"fmt"
	"sync"
)

// Shard structure to hold blocks
type Shard struct {
	ID     int
	Blocks []*Block
	mu     sync.Mutex
}

// Initialize shards
var shards []Shard // Use exported NumShards

func initShards() {
	shards = make([]Shard, NumShards) 
	for i := 0; i < NumShards; i++ {
		shards[i] = Shard{ID: i}
	}
}

// Assign blocks to shards based on their ShardID
func distributeBlocksToShards() {
	// Clear previous assignments
	for i := range shards {
		shards[i].Blocks = nil
	}

	BlockchainMu.Lock() // ✅ Use the exported BlockchainMu
	defer BlockchainMu.Unlock()

	for i := range Blockchain { 
		block := &Blockchain[i]
		shardID := block.ShardID
		if shardID >= NumShards || shardID < 0 {
			fmt.Printf("⚠️ Invalid Shard ID: %d for Block %d\n", shardID, block.Index)
			continue
		}
		shards[shardID].mu.Lock()
		shards[shardID].Blocks = append(shards[shardID].Blocks, block)
		shards[shardID].mu.Unlock()
	}
}

// Visualize shards and blocks
func visualizeShards() {
	distributeBlocksToShards()

	fmt.Println("\n🛠 Blockchain Sharding Visualization:")
	fmt.Println("=====================================")
	for i := range shards {
		shards[i].mu.Lock()
		fmt.Printf("🟢 Shard %d (%d blocks):\n", shards[i].ID, len(shards[i].Blocks))
		fmt.Println("-------------------------")

		for _, block := range shards[i].Blocks {
			fmt.Printf("| 🧩 Block %d (Shard %d | Hash: %.6s) |\n", block.Index, block.ShardID, block.Hash)
			for _, tx := range block.Transactions {
				fmt.Printf("   ↳ 🔄 Tx: %s (ContainerID: %s)\n", tx.TransactionID, tx.ContainerID)
			}
		}
		if len(shards[i].Blocks) == 0 {
			fmt.Println("| 🚫 No blocks assigned |")
		}
		fmt.Print("-------------------------\n")
		shards[i].mu.Unlock()
	}
}

// Get blocks belonging to a specific shard
func getShardBlocks(shardID int) []*Block {
	if shardID < 0 || shardID >= NumShards {
		fmt.Printf("⚠️ Invalid Shard ID: %d\n", shardID)
		return nil
	}
	shards[shardID].mu.Lock()
	defer shards[shardID].mu.Unlock()
	return shards[shardID].Blocks
}
