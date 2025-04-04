package main

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var firestoreClient *firestore.Client

func InitFirebase() {
	ctx := context.Background()
	opt := option.WithCredentialsFile("serviceAccountKey.json")

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
		"timestamp": tx.Timestamp,
	})

	if err != nil {
		log.Printf("❌ Failed to write transaction to Firestore: %v", err)
	} else {
		log.Println("✅ Transaction saved to Firestore")
	}
}

// Load a specific sytem after finishing
func LoadTransactionsFromFirestore() []TransactionLog {
	ctx := context.Background()
	var logs []TransactionLog

	iter := firestoreClient.Collection("transactions").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}

		var tx TransactionLog
		data := doc.Data()

		tx.TxID = data["txID"].(string)
		tx.Source = int(data["source"].(int64)) // Firestore numbers are int64
		tx.Target = int(data["target"].(int64))
		tx.Message = data["message"].(string)
		tx.Type = data["type"].(string)
		tx.ExecTime = data["execTime"].(float64)
		tx.Timestamp = data["timestamp"].(string)

		logs = append(logs, tx)
	}

	log.Printf("📦 Loaded %d transactions from Firestore\n", len(logs))
	return logs
}
