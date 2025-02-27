Blockchain Concurrency Confliction Resolution

Go-Based backend code, paired with a Python front-end

> Project Overview

This project is a blockchain-based concurrency conflict resolution system designed to handle transaction conflicts efficiently. It integrates a Go-based backend for high-performance blockchain processing and a Python-based front-end for user interaction and visualization.

> Project Structure

Blockchain-Concurrency-Confliction-Resolution/  
│── Blockchain_Codebase/      # Go-based blockchain implementation  
│── backup/                   # Backup of essential blockchain assets  
│── chaincode/                 # Hyperledger Fabric smart contracts  
│── fablo-target/              # Fablo-generated network artifacts  
│── fabric-samples/            # Hyperledger Fabric sample configurations  
│── GUI.py                     # Python GUI front-end  
│── blockchain_visualisation.html  # HTML-based blockchain visualization  
│── connection-profile.yaml     # Hyperledger Fabric network connection profile  
│── fablo-config.json           # Configuration file for Fablo  
│── go.mod                      # Go module dependencies  
│── go.sum                      # Go module hash checksums  
│── orderer-ca.crt              # Certificate for orderer node  
│── org2-ca.crt                 # Certificate for Org2 peer node  
└── README.md                   # Documentation (this file)  

Features

✅ Blockchain Implementation: Utilizes Go to implement and manage blockchain transactions.  
✅ Concurrency Resolution: Efficiently handles transaction conflicts using optimized algorithms.  
✅ ReactFlow-Based Visualization: Provides an interactive blockchain visualization using ReactFlow.  
✅ Hyperledger Fabric Integration: Uses chaincode for executing business logic on a private blockchain network.  
✅ Python GUI: Simple GUI for managing and testing transactions.  
✅ Backup & Recovery: Includes a backup mechanism to recover lost blockchain states.  

> **Installation & Setup**

🔹 Prerequisites
Ensure you have the following installed on your system:

Go (>=1.23.2)

Node.js & npm (for front-end)

Python (for GUI interaction)

Docker & Fabric Tools (for Hyperledger Fabric)

WSL2 (if running on Windows)
🔹 Cloning the Repository

git clone https://github.com/ConnorM2020/Blockchain-Concurrency-Confliction-Resolution.git

cd Blockchain-Concurrency-Confliction-Resolution

🔹 Running the Blockchain Network

cd Blockchain_Codebase
./startFabric.sh   # Starts the Hyperledger Fabric network

🔹 Running the Go Backend

cd Blockchain_Codebase
go run main.go

🔹 Running the ReactFlow Front-End

cd Blockchain_Codebase/blockchain-visualizer
npm install
npm run dev

(Include images of the system, architecture, or UI here)

> **Contributors**
ConnorM2020 

License

This project is licensed under the MIT License. Feel free to modify and use it in your projects.
Contact & Support

For issues and contributions, create a GitHub issue or reach out via: 
GitHub: ConnorM2020
