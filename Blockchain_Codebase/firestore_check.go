// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"cloud.google.com/go/firestore"
// 	"google.golang.org/api/iterator"
// 	"google.golang.org/api/option"
// )

// func main() {
// 	ctx := context.Background()

// 	client, err := firestore.NewClient(ctx, "csc4006-blockchain", option.WithCredentialsFile(
// 		"/mnt/c/Users/cmall/Documents/MENG_CSC/Level_4/CSC4006_Project/concurrency-confliction/40295919/Blockchain_Codebase/serviceAccountKey.json",
// 	))
// 	if err != nil {
// 		log.Fatalf("❌ Firestore client init failed: %v", err)
// 	}
// 	defer client.Close()

// 	// Outer collection
// 	outerIter := client.Collection("transactions").Documents(ctx)
// 	defer outerIter.Stop()

// 	total := 0
// 	for {
// 		parentDoc, err := outerIter.Next()
// 		if err == iterator.Done {
// 			break
// 		}
// 		if err != nil {
// 			log.Printf("❌ Failed to get parent document: %v", err)
// 			continue
// 		}

// 		subIter := parentDoc.Ref.Collection("transactions").Documents(ctx)
// 		defer subIter.Stop()

// 		for {
// 			subDoc, err := subIter.Next()
// 			if err == iterator.Done {
// 				break
// 			}
// 			if err != nil {
// 				log.Printf("❌ Error reading transaction document: %v", err)
// 				break
// 			}

// 			fmt.Printf("✅ Transaction Doc ID: %s (Parent: %s)\n", subDoc.Ref.ID, parentDoc.Ref.ID)
// 			total++
// 		}
// 	}

// 	fmt.Printf("📦 Total nested transaction documents fetched: %d\n", total)
// }
