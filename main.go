package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"time"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

const (
	walletPath        = "wallet"
	connectionProfile = "connection-profile.yaml"
	chaincodeName     = "mychaincode"
	channelName       = "mychannel"
	orgMSP            = "Org1MSP"
	adminIdentity     = "Admin"
)

func init() {
	// Enable debug logging for Hyperledger Fabric SDK
	logging.SetLevel("fabsdk", logging.DEBUG)
	logging.SetLevel("gateway", logging.DEBUG)

	// Seed the random generator for delay simulation
	rand.Seed(time.Now().UnixNano())
}
func addAdminIdentityToWallet(wallet *gateway.Wallet) error {
	log.Println("[Admin Setup] Adding Admin identity to wallet.")

	certPath := filepath.Join(walletPath, "signcerts", "cert.pem")
	keyPath := filepath.Join(walletPath, "keystore", "0974853793a9492bc021a8453e71d3d49a4173ff1ef01b2e298b515b93f56686_sk")

	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read cert.pem: %w", err)
	}

	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	identity := gateway.NewX509Identity(orgMSP, string(cert), string(key))
	err = wallet.Put(adminIdentity, identity)
	if err != nil {
		return fmt.Errorf("failed to add Admin identity to wallet: %w", err)
	}
	log.Println("[Admin Setup] Admin identity successfully added to wallet.")
	return nil
}

func writeToKey(clientID, key, value string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Load wallet
	wallet, err := gateway.NewFileSystemWallet(walletPath)
	if err != nil {
		log.Printf("[%s] Failed to create wallet: %v", clientID, err)
		return
	}

	// Check if Admin identity exists in the wallet
	if !wallet.Exists(adminIdentity) {
		log.Printf("[%s] Admin identity not found in wallet. Adding identity.", clientID)

		err := addAdminIdentityToWallet(wallet)
		if err != nil {
			log.Printf("[%s] Failed to add Admin identity: %v", clientID, err)
			return
		}
	}
	// Connect to the gateway
	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(connectionProfile))),
		gateway.WithIdentity(wallet, adminIdentity),
	)
	if err != nil {
		log.Printf("[%s] Failed to connect to gateway: %v", clientID, err)
		return
	}
	defer gw.Close()

	// Access the network and contract
	network, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Printf("[%s] Failed to get network: %v", clientID, err)
		return
	}
	contract := network.GetContract(chaincodeName)
	// Submit transaction
	_, err = contract.SubmitTransaction("PutState", key, value)
	if err != nil {
		log.Printf("[%s] Failed to submit transaction: %v", clientID, err)
		return
	}
	log.Printf("[%s] Successfully wrote key: %s with value: %s", clientID, key, value)
}

func main() {
	var wg sync.WaitGroup

	// Simulate two clients writing to the same key
	key := "conflictKey"
	client1Value := "ValueFromClient1"
	client2Value := "ValueFromClient2"
	wg.Add(2)
	go writeToKey("Client1", key, client1Value, &wg)
	go writeToKey("Client2", key, client2Value, &wg)
	wg.Wait()
	fmt.Println("Both clients attempted to write to the same key.")
}
