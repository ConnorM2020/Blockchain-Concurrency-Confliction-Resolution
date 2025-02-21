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
CONCURRENCY_API_URL = "http://localhost:8080/conflicts"
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

        if not response:
            logging.error("Transaction processing returned no response.")
            return jsonify({"message": "Transaction queued but not confirmed yet.", "status": "pending"}), 202

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

        # Ensure response content exists before using `.json()`
        response_data = response.json() if response.content else {}

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

        elif response.status_code == 409:  # Conflict
            conflict_message = {
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
                "message": f"⚠️ Concurrency conflict detected! Transaction already exists for container: {transaction['containerID']}",
                "containerID": transaction['containerID']
            }

            logging.warning(conflict_message["message"])
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
                
                // Wait 2 seconds before fetching transactions again
                setTimeout(fetchTransactions, 2000);
            } catch (error) {
                console.error("Transaction failed:", error);
                alert(`Transaction failed: ${error.message}`);
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
                        const status = tx.version === -1 ? "failed" : "completed";  // If version is -1, it failed

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
       let conflictPolling = true; // Control polling state

       async function fetchConflicts() {
            try {
                const response = await fetch('/conflicts');
                const data = await response.json();
                const conflictContainer = document.getElementById("conflict_list");

                if (data.total_conflicts === 0) {
                    conflictContainer.innerHTML = "<p style='color: green; font-weight: bold;'>✅ No concurrency conflicts detected.</p>";

                    // 🛑 Stop polling if there are no conflicts
                    if (conflictPolling) {
                        console.log("✅ No conflicts detected. Stopping conflict polling.");
                        conflictPolling = false;
                    }
                    return;
                }

                // Reactivate polling if new conflicts appear
                conflictPolling = true;

                // Display conflicts
                conflictContainer.innerHTML = data.conflicts.map(conflict =>
                    `<div style="color: red; font-weight: bold;"><strong>[${new Date().toLocaleString()}]</strong> ${conflict.message}</div>`
                ).join("");

            } catch (error) {
                console.error("❌ Error fetching concurrency conflicts:", error);
            }
        }

        // Poll only if conflicts are present
        setInterval(() => {
            if (conflictPolling) {
                fetchConflicts();
            }
        }, 5000);
        
        async function processTransaction(containerID, attempt = 0) {
            try {
                const response = await fetch('/add_transaction', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ containerID })
                });

                const data = await response.json();

                if (response.ok) {
                    alert("✅ Transaction added successfully!");

                    // 🛠 Update UI immediately
                    fetchTransactions();
                    setTimeout(fetchConflicts, 2000);
                    return;
                } 

                if (response.status === 409) { // Concurrency Conflict
                    console.warn(`⚠️ Conflict detected! Retrying in ${500 * (2 ** attempt)}ms...`);

                    if (attempt < 3) {
                        const delay = (500 * (2 ** attempt)); // Exponential Backoff
                        await new Promise(res => setTimeout(res, delay));
                        return processTransaction(containerID, attempt + 1);
                    } else {
                        console.error("❌ Transaction failed due to concurrency conflicts.");
                        alert("❌ Transaction failed due to concurrency conflicts.");
                        fetchConflicts();  // ✅ Fetch conflicts if transaction fails
                        return;
                    }
                } 
                throw new Error(data.error || "Unknown error");

            } catch (error) {
                console.error("Transaction error:", error);
                alert(`Error processing transaction: ${error.message}`);
            }
        }
        window.onload = () => {
            fetchTransactions();
            fetchConflicts(); // Load conflicts immediately on page load
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
        # Fetch blockchain data from Go API
        response = requests.get(CONCURRENCY_API_URL)

        # Check if response is successful
        if response.status_code != 200:
            logging.error(f"Failed to fetch conflicts: {response.status_code}, Response: {response.text}")
            return jsonify({"error": f"Blockchain API returned {response.status_code}", "details": response.text}), 500

        # Check if response is JSON
        try:
            blockchain_data = response.json()
        except requests.exceptions.JSONDecodeError:
            logging.error("Invalid JSON received from blockchain API")
            return jsonify({"error": "Invalid JSON response from blockchain API"}), 500

        # Validate the expected format (it should be a dictionary)
        if not isinstance(blockchain_data, dict):
            logging.error("Blockchain API returned unexpected format")
            return jsonify({"error": "Unexpected blockchain response format"}), 500

        # Fix potential 'null' conflicts field issue
        conflicts = blockchain_data.get("conflicts") or []

        if not isinstance(conflicts, list):
            logging.error("Blockchain API returned an invalid conflicts format")
            return jsonify({"error": "Invalid conflicts data format"}), 500

        logging.info(f"Detected {len(conflicts)} concurrency conflicts")
        return jsonify({"total_conflicts": len(conflicts), "conflicts": conflicts})

    except requests.exceptions.RequestException as e:
        logging.error(f"Blockchain API request failed: {e}")
        return jsonify({"error": "Blockchain API request failed", "details": str(e)}), 500


# Store concurrency conflicts
#concurrency_conflicts = []
def process_transaction(transaction_id):
    """Process a transaction with retry mechanism and optimistic concurrency control."""
    MAX_RETRIES = 3
    transaction = transaction_queue.get(transaction_id)

    if not transaction:
        logging.error(f"❌ Transaction {transaction_id} not found in queue.")
        return jsonify({"error": "Transaction not found"}), 404

    try:
        transaction['retry_count'] += 1
        logging.info(f"🔄 Processing transaction {transaction_id}, Attempt {transaction['retry_count']}")

        # Log what is being sent
        transaction_payload = {
            "containerID": transaction['containerID'],
            "transactionID": transaction_id,
            "version": transaction['version']
        }
        logging.debug(f"📤 Sending transaction to {ADD_TRANSACTION_API_URL}: {transaction_payload}")

        # Send transaction to Go API
        response = requests.post(ADD_TRANSACTION_API_URL, json=transaction_payload)

        # Log response
        logging.debug(f"📥 Received response {response.status_code}: {response.text}")

        if response.status_code == 201:
            transaction['status'] = 'completed'
            transaction['version'] += 1
            logging.info(f"✅ Transaction {transaction_id} successful")
            return jsonify({
                "message": "Transaction added successfully",
                "transactionID": transaction_id,
                "version": transaction['version']
            }), 201

        elif response.status_code == 409:  # Conflict
            logging.warning(f"⚠️ Conflict detected for transaction {transaction_id}")

            if transaction['retry_count'] < MAX_RETRIES:
                time.sleep(0.5 * (2 ** transaction['retry_count']))  # Exponential Backoff
                return process_transaction(transaction_id)
            else:
                transaction['status'] = 'failed'
                logging.error(f"❌ Transaction {transaction_id} failed after {MAX_RETRIES} retries")
                return jsonify({
                    "error": "Transaction failed due to concurrency conflicts",
                    "transactionID": transaction_id
                }), 409

        else:
            logging.error(f"🚨 Unexpected response: {response.status_code}, Details: {response.text}")
            return jsonify({
                "error": "Transaction processing failed",
                "details": response.text
            }), response.status_code

    except requests.exceptions.RequestException as e:
        logging.error(f"❌ Blockchain API request failed: {e}")
        return jsonify({"error": "Blockchain API unreachable"}), 500

# **Concurrency Conflict Fetch Endpoint**
@app.route('/conflicts', methods=['GET'])
def get_conflicts():
    """Fetch stored concurrency conflicts."""
    logging.info(f"Fetching concurrency conflicts. Total: {len(concurrency_conflicts)}")

    return jsonify({
        "total_conflicts": len(concurrency_conflicts),
        "conflicts": concurrency_conflicts
    })

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
