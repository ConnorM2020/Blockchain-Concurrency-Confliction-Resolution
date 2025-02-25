package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an asset
type SmartContract struct {
	contractapi.Contract
}

// InitLedger initializes the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// CreateAsset creates a new asset
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, value string) error {
	return ctx.GetStub().PutState(id, []byte(value))
}

// ReadAsset returns the asset stored in the ledger
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	value, err := ctx.GetStub().GetState(id)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if value == nil {
		return "", fmt.Errorf("asset not found: %s", id)
	}
	return string(value), nil
}

func main() {
	cc, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating chaincode: %s", err)
	}
	if err := cc.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
