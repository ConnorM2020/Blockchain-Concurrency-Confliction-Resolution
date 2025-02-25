package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// AssetTransfer Smart Contract
type AssetTransfer struct {
	contractapi.Contract
}

// InitLedger initializes the ledger with sample data
func (t *AssetTransfer) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Ledger initialized for asset transfer")
	return nil
}

func main() {
	cc, err := contractapi.NewChaincode(new(AssetTransfer))
	if err != nil {
		fmt.Printf("Error creating asset transfer chaincode: %s", err)
		return
	}

	if err := cc.Start(); err != nil {
		fmt.Printf("Error starting asset transfer chaincode: %s", err)
	}
}
