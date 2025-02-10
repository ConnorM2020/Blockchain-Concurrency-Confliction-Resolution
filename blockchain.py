from flask import Flask, jsonify, render_template_string, request
import requests
import logging
import time
import uuid

app = Flask(__name__)

# Enable logging for debugging
logging.basicConfig(level=logging.INFO)

# Go Blockchain API Endpoints
BLOCKCHAIN_API_URL = "http://localhost:8080/blockchain"
ADD_BLOCK_API_URL = "http://localhost:8080/addBlock"
ADD_TRANSACTION_API_URL = "http://localhost:8080/addTransaction"


# In-memory transaction queue with versioning
transaction_queue = {}
concurrency_conflicts = []

# **Transaction Addition Endpoint**
@app.route('/add_transaction', methods=['POST'])
def add_transaction():
    """Sends a new transaction to the Go blockchain API with concurrency control."""
    try:
        data = request.get_json()
        if not data or "containerID" not in data:
            return jsonify({"error": "Missing 'containerID' in request"}), 400

        # Generate a unique transaction ID and timestamp
        transaction_id = str(uuid.uuid4())
        timestamp = int(time.time())

        # Create transaction entry with version control
        transaction_entry = {
            "id": transaction_id,
            "containerID": data["containerID"],
            "timestamp": timestamp,
            "status": "pending",
            "version": 0,
            "retry_count": 0
        }

        # Add to transaction queue
        transaction_queue[transaction_id] = transaction_entry

        # Attempt to add transaction
        response = process_transaction(transaction_id)

        # Ensure response is returned
        if not response:
            return jsonify({"error": "Transaction processing returned no response"}), 500

        return response

    except Exception as e:
        logging.error(f"Transaction addition error: {e}")
        return jsonify({"error": "Transaction processing failed"}), 500


def process_transaction(transaction_id):
    """Process a transaction with retry mechanism and optimistic concurrency control."""
    MAX_RETRIES = 3
    transaction = transaction_queue.get(transaction_id)

    if not transaction:
        return jsonify({"error": "Transaction not found"}), 404
    try:
        # Increment retry count
        transaction['retry_count'] += 1

        # Send transaction to Go API
        response = requests.post(ADD_TRANSACTION_API_URL, json={
            "containerID": transaction['containerID'],
            "transactionID": transaction_id,
            "version": transaction['version']
        })

        # Check API response
        if response.status_code == 201:
            # Successful transaction
            transaction['status'] = 'completed'
            transaction['version'] += 1
            logging.info(f"✅ Transaction {transaction_id} successful")
            return jsonify({
                "message": "Transaction added successfully",
                "transactionID": transaction_id,
                "version": transaction['version']
            }), 201

        elif response.status_code == 409:  # Conflict
            # Log the conflict
            conflict_message = {
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
                "message": f"⚠️ Concurrency conflict detected! Transaction already exists for container: {transaction['containerID']}",
                "containerID": transaction['containerID']
            }

            logging.warning(conflict_message["message"])

            # Store conflict if it's not already recorded
            if conflict_message not in concurrency_conflicts:
                concurrency_conflicts.append(conflict_message)

            if transaction['retry_count'] < MAX_RETRIES:
                # Retry with exponential backoff
                time.sleep(0.5 * (2 ** transaction['retry_count']))
                return process_transaction(transaction_id)
            else:
                # Max retries reached
                transaction['status'] = 'failed'
                logging.error(f"❌ Transaction {transaction_id} failed after {MAX_RETRIES} retries")
                return jsonify({
                    "error": "Transaction failed due to concurrency conflicts",
                    "transactionID": transaction_id
                }), 409

        else:
            # Other unexpected errors
            transaction['status'] = 'failed'
            logging.error(f"Unexpected error: {response.text}")
            return jsonify({
                "error": "Transaction processing failed",
                "details": response.text
            }), response.status_code

    except requests.exceptions.RequestException as e:
        logging.error(f"Blockchain API request failed: {e}")
        return jsonify({"error": "Blockchain API unreachable"}), 500



# **Blockchain Fetch Endpoint**
@app.route('/fetch_blocks', methods=['GET'])
def fetch_blocks():
    """Fetches the blockchain from the Go API."""
    try:
        logging.info(f"Fetching blockchain from {BLOCKCHAIN_API_URL}")
        response = requests.get(BLOCKCHAIN_API_URL)
        response.raise_for_status()

        data = response.json()
        logging.info(f"Blockchain data received: {data}")

        if not isinstance(data, list):
            logging.error("Invalid response format from blockchain API")
            return jsonify({"error": "Invalid blockchain response"}), 500

        return jsonify(data)

    except requests.exceptions.ConnectionError:
        logging.error("Cannot connect to Blockchain API. Ensure Go server is running.")
        return jsonify({"error": "Blockchain API unreachable. Is the Go server running?"}), 500

    except requests.exceptions.JSONDecodeError:
        logging.error("Received invalid JSON response from the Blockchain API.")
        return jsonify({"error": "Invalid JSON response from blockchain API"}), 500

    except requests.exceptions.RequestException as e:
        logging.error(f"Blockchain API request failed: {e}")
        return jsonify({"error": "Blockchain API request failed"}), 500



# **HTML for the Main Dashboard**
MAIN_PAGE = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Blockchain & Transactions</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; background-color: #f9f9f9; padding: 50px; }
        h1 { color: #333; }
        button { padding: 10px 15px; font-size: 16px; margin: 10px; border: none; border-radius: 5px; background-color: #007bff; color: white; cursor: pointer; }
        button:hover { background-color: #0056b3; }
    </style>
</head>
<body>
    <h1>Blockchain & Transactions Dashboard</h1>
    <button onclick="openBlockchain()">Open Blockchain Viewer</button>
    <button onclick="openTransactions()">Open Transactions Window</button>

    <script>
        function openBlockchain() { window.open('/blockchain', '_blank'); }
        function openTransactions() { window.open('/transactions', '_blank'); }
    </script>
</body>
</html>
"""

# **HTML for the Blockchain Visualizer**
BLOCKCHAIN_PAGE = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Blockchain Viewer</title>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f9f9f9; text-align: center; padding: 20px; }
        h1 { color: #333; }
        .block { background: white; padding: 15px; margin: 10px auto; border-radius: 10px; width: 80%; max-width: 600px; box-shadow: 0px 4px 6px rgba(0, 0, 0, 0.1); text-align: left; }
        .back-button { display: inline-block; margin: 20px; padding: 10px 15px; background: #007bff; color: white; border: none; border-radius: 5px; cursor: pointer; }
        .back-button:hover { background: #0056b3; }
    </style>
    <script>
        function fetchBlockchain() {
            fetch('http://127.0.0.1:5000/fetch_blocks')
                .then(response => response.json())
                .then(data => {
                    const container = document.getElementById("blockchain");
                    container.innerHTML = "";

                    if (!Array.isArray(data) || data.length === 0) {
                        container.innerHTML = "<p>No blockchain data available.</p>";
                        return;
                    }

                    data.forEach(block => {
                        const div = document.createElement("div");
                        div.className = "block";
                        div.innerHTML = `
                            <strong>Block Index:</strong> ${block.index}<br>
                            <strong>Container ID:</strong> ${block.container_id || 'N/A'}<br>
                            <strong>Hash:</strong> ${block.hash}<br>
                            <strong>Previous Hash:</strong> ${block.previous_hash}<br>
                            <strong>Version:</strong> ${block.version}<br>
                            <strong>Transactions:</strong>
                            <ul>
                                ${block.transactions.map(tx => 
                                    `<li>ContainerID: ${tx.container_id}, Timestamp: ${tx.timestamp}</li>`
                                ).join('')}
                            </ul>
                        `;
                        container.appendChild(div);
                    });
                })
                .catch(error => {
                    console.error("Error fetching blockchain:", error);
                    document.getElementById("blockchain").innerHTML = "<p>Error fetching blockchain data.</p>";
                });
        }
        window.onload = fetchBlockchain;
    </script>

</head>
<body>
    <h1>Blockchain Viewer</h1>
    <button class="back-button" onclick="window.location.href='/'">Back to Dashboard</button>
    <div id="blockchain">
        <p>Loading blockchain data...</p>
    </div>
</body>
</html>
"""

# **HTML for the Transactions Window**
TRANSACTION_PAGE = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Transaction Manager</title>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f9f9f9; text-align: center; padding: 20px; }
        h1 { color: #333; }
        .container { max-width: 800px; margin: auto; background: white; padding: 20px; border-radius: 10px; box-shadow: 0px 4px 6px rgba(0, 0, 0, 0.1); }
        input { width: 80%; padding: 10px; margin: 10px 0; border: 1px solid #ccc; border-radius: 5px; }
        button { padding: 10px 15px; background: #007bff; color: white; border: none; border-radius: 5px; cursor: pointer; }
        button:hover { background: #0056b3; }
        .transaction-list { margin-top: 20px; text-align: left; }
        .pending { color: orange; font-weight: bold; }
        .completed { color: green; font-weight: bold; }
        .failed { color: red; font-weight: bold; }
        .conflict-list { margin-top: 20px; text-align: left; color: red; font-weight: bold; }
    </style>
    <script>
        let pendingTransactions = {}; // Track pending transactions

        async function addTransaction() {
            const containerID = document.querySelector("#container_id").value.trim();

            if (!containerID) {
                alert("Please enter a Container ID.");
                return;
            }

            if (pendingTransactions[containerID]) {
                alert("Transaction is already in progress. Please wait.");
                return;
            }

            pendingTransactions[containerID] = { status: "pending", retries: 0 };

            try {
                await processTransaction(containerID);
            } catch (error) {
                console.error("Transaction failed:", error);
                alert(`Transaction failed: ${error.message}`);
            }
        }

        async function processTransaction(containerID, attempt = 0) {
            try {
                const response = await fetch('/add_transaction', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ containerID })
                });

                const data = await response.json();

                if (response.ok) {
                    pendingTransactions[containerID].status = "completed";
                    alert("✅ Transaction added successfully!");
                   
                    fetchTransactions();
                } else if (response.status === 409) { // Concurrency Conflict

                    fetchConflicts();

                    if (attempt < 3) {
                        const delay = (500 * (2 ** attempt)); // Exponential Backoff
                        console.warn(`⚠️ Conflict detected! Retrying in ${delay}ms...`);
                        await new Promise(res => setTimeout(res, delay));
                        return processTransaction(containerID, attempt + 1);
                    } else {
                        pendingTransactions[containerID].status = "failed";
                        alert("❌ Transaction failed due to concurrency conflicts.");
                    }
                } else {
                    throw new Error(data.error || "Unknown error");
                }
            } catch (error) {
                console.error("Transaction error:", error);
                alert(`Error processing transaction: ${error.message}`);
            } finally {
                delete pendingTransactions[containerID]; // Clean up
            }
        }

        async function fetchTransactions() {
            try {
                const response = await fetch('/fetch_blocks');
                const data = await response.json();
                const transactionContainer = document.getElementById("transaction_list");
                transactionContainer.innerHTML = "";

                if (!Array.isArray(data) || data.length === 0) {
                    transactionContainer.innerHTML = "<p>No transactions available.</p>";
                    return;
                }

                data.forEach(block => {
                    block.transactions.forEach(tx => {
                        const div = document.createElement("div");
                        const status = pendingTransactions[tx.container_id]?.status || "completed";
                        div.className = `transaction ${status}`;
                        div.innerHTML = `
                            <strong>Container ID:</strong> ${tx.container_id} | 
                            <strong>Timestamp:</strong> ${tx.timestamp} | 
                            <span class="${status}">${status.toUpperCase()}</span>
                        `;
                        transactionContainer.appendChild(div);
                    });
                });
            } catch (error) {
                console.error("Error fetching transactions:", error);
            }
        }
        async function fetchConflicts() {
            try {
                const response = await fetch('/conflicts');
                const data = await response.json();
                const conflictContainer = document.getElementById("conflict_list");
                conflictContainer.innerHTML = "";

                if (data.total_conflicts === 0) {
                    conflictContainer.innerHTML = "<p>No concurrency conflicts detected.</p>";
                    return;
                }

                data.conflicts.forEach(conflict => {
                    const div = document.createElement("div");
                    div.innerHTML = `<strong>[${conflict.timestamp}]</strong> ${conflict.message}`;
                    conflictContainer.appendChild(div);
                });

            } catch (error) {
                console.error("Error fetching concurrency conflicts:", error);
            }
        }
    window.onload = () => {
        fetchTransactions();
        fetchConflicts();
    };

    setInterval(fetchConflicts, 5000); // Refresh conflicts every 5 seconds
    </script>
</head>
<body>
    <h1>Transaction Manager</h1>
    <div class="container">
        <h2>Add Transaction</h2>
        <input type="text" id="container_id" placeholder="Enter Container ID">
        <button onclick="addTransaction()">Submit Transaction</button>
        <button onclick="window.location.href='/'">Back to Dashboard</button>
        <button onclick="fetchConflicts()">Check Concurrency</button>

        <h2>Concurrency Conflicts</h2>
        <div id="conflict_list" class="conflict-list"></div>

        <h2>Transaction History</h2>
        <div id="transaction_list" class="transaction-list"></div>
    </div>
</body>
</html>

"""

# **Conflict Detection Endpoint**
@app.route('/conflicts', methods=['GET'])
def detect_conflicts():
    """Detects concurrency conflicts in the blockchain data."""
    try:
        # Fetch blockchain data
        response = requests.get(BLOCKCHAIN_API_URL)
        response.raise_for_status()
        blockchain_data = response.json()

        if not isinstance(blockchain_data, list):
            return jsonify({"error": "Invalid blockchain response format"}), 500

        # Detect concurrency conflicts (Example: Duplicate containerID in different blocks)
        seen_containers = {}
        conflicts = []

        for block in blockchain_data:
            for tx in block.get("transactions", []):
                container_id = tx.get("container_id")

                if container_id in seen_containers:
                    conflicts.append(f"Conflict detected: Container {container_id} appears in multiple transactions.")

                seen_containers[container_id] = block["index"]

        return jsonify({"total_conflicts": len(conflicts), "conflicts": conflicts})

    except requests.exceptions.RequestException as e:
        return jsonify({"error": "Failed to fetch blockchain data", "details": str(e)}), 500
# Store concurrency conflicts
concurrency_conflicts = []

def process_transaction(transaction_id):
    """Process a transaction with retry mechanism and optimistic concurrency control."""
    MAX_RETRIES = 3
    transaction = transaction_queue.get(transaction_id)

    if not transaction:
        return jsonify({"error": "Transaction not found"}), 404

    try:
        # Increment retry count
        transaction['retry_count'] += 1

        # Send transaction to Go API
        response = requests.post(ADD_TRANSACTION_API_URL, json={
            "containerID": transaction['containerID'],
            "transactionID": transaction_id,
            "version": transaction['version']
        })

        # Check API response
        if response.status_code == 201:
            transaction['status'] = 'completed'
            transaction['version'] += 1
            logging.info(f"✅ Transaction {transaction_id} successful")
            return jsonify({
                "message": "Transaction added successfully",
                "transactionID": transaction_id,
                "version": transaction['version']
            }), 201

        # call to working_main.go, block.go 
        elif response.status_code == 409:  # Conflict Detected
            conflict_message = {
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
                "message": f"⚠️ Concurrency conflict detected! Transaction already exists for container: {transaction['containerID']}",
                "containerID": transaction['containerID']
            }

            logging.warning(conflict_message["message"])

            # Store conflict only if it's not already recorded
            if conflict_message not in concurrency_conflicts:
                concurrency_conflicts.append(conflict_message)

            if transaction['retry_count'] < MAX_RETRIES:
                time.sleep(0.5 * (2 ** transaction['retry_count']))  # Exponential backoff
                return process_transaction(transaction_id)
            else:
                transaction['status'] = 'failed'
                logging.error(f"❌ Transaction {transaction_id} failed after {MAX_RETRIES} retries")
                return jsonify({
                    "error": "Transaction failed due to concurrency conflicts",
                    "transactionID": transaction_id
                }), 409

        else:
            transaction['status'] = 'failed'
            logging.error(f"Unexpected error: {response.text}")
            return jsonify({
                "error": "Transaction processing failed",
                "details": response.text
            }), response.status_code

    except requests.exceptions.RequestException as e:
        logging.error(f"Blockchain API request failed: {e}")
        return jsonify({"error": "Blockchain API unreachable"}), 500

    
# **Concurrency Conflict Fetch Endpoint**
@app.route('/conflicts', methods=['GET'])
def get_conflicts():
    """Fetch concurrency conflicts."""
    return jsonify({
        "total_conflicts": len(concurrency_conflicts),
        "conflicts": concurrency_conflicts})


@app.route('/')
def home():
    return render_template_string(MAIN_PAGE)

@app.route('/transactions')
def transactions_page():
    return render_template_string(TRANSACTION_PAGE)

@app.route('/blockchain')
def blockchain_page():
    return render_template_string(BLOCKCHAIN_PAGE)

if __name__ == '__main__':
    app.run(debug=True)
