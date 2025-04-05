package main

import (
	"context"
	"log"
	"os"
	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var firestoreClient *firestore.Client

func InitFirebase() {
	ctx := context.Background()

	// Read credentials manually
	credsPath := "/mnt/c/Users/cmall/Documents/MENG_CSC/Level_4/CSC4006_Project/concurrency-confliction/40295919/Blockchain_Codebase/serviceAccountKey.json"
	credsJSON, err := os.ReadFile(credsPath)
	if err != nil {
		log.Fatalf("❌ Could not read service account key file: %v", err)
	}

	// Use the JSON directly
	opt := option.WithCredentialsJSON(credsJSON)

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: "csc4006-blockchain"}, opt)
	if err != nil {
		log.Fatalf("🔥 Failed to initialize Firebase: %v", err)
	}
	log.Println("✅ Firebase initialized")

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to init Firestore client: %v", err)
	}
	firestoreClient = client
	log.Println("📦 Firestore client ready")
}


func SaveTransactionToFirestore(tx TransactionLog) {
	ctx := context.Background()

	_, _, err := firestoreClient.Collection("transactions").Add(ctx, map[string]interface{}{
		"txID":      tx.TxID,
		"source":    tx.Source,
		"target":    tx.Target,
		"message":   tx.Message,
		"type":      tx.Type,
		"execTime":  tx.ExecTime,
		"finality":  tx.Finality,
		"timestamp": tx.Timestamp,
		"propagationLatency": tx.Propagation,
		"tps":       tx.TPS,
	})

	if err != nil {
		log.Printf("❌ Failed to write transaction to Firestore: %v", err)
	} else {
		log.Println("✅ Transaction saved to Firestore")
	}
}
func LoadTransactionsFromFirestore() []TransactionLog {
	ctx := context.Background()
	var logs []TransactionLog

	iter := firestoreClient.Collection("transactions").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("❌ Error reading document: %v", err)
			continue
		}

		data := doc.Data()
		var tx TransactionLog

		if val, ok := data["txID"].(string); ok {
			tx.TxID = val
		} else {
			continue
		}

		if val, ok := data["source"].(int64); ok {
			tx.Source = int(val)
		}

		if val, ok := data["target"].(int64); ok {
			tx.Target = int(val)
		}

		if val, ok := data["message"].(string); ok {
			tx.Message = val
		}

		if val, ok := data["type"].(string); ok {
			tx.Type = val
		}

		if val, ok := data["execTime"].(float64); ok {
			tx.ExecTime = val
		}

		if val, ok := data["finality"].(float64); ok {
			tx.Finality = val
		}

		if val, ok := data["propagationLatency"].(float64); ok {
			tx.Propagation = val
		}

		if val, ok := data["tps"].(float64); ok {
			tx.TPS = val
		}

		if val, ok := data["timestamp"].(string); ok {
			tx.Timestamp = val
		}

		logs = append(logs, tx)
	}

	log.Printf("📦 Loaded %d transactions from Firestore\n", len(logs))
	return logs
}
