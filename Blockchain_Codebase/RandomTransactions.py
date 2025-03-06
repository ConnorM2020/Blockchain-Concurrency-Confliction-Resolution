import requests
import random
import string
import threading

# Stress-test the Blockchain network
API_BASE = "http://localhost:8080"

def generate_large_data(size=5000):
    """Generate a large random string of a given size."""
    return ''.join(random.choices(string.ascii_letters + string.digits, k=size))

def send_transaction():
    """Send a single large transaction."""
    payload = {
        "source": random.randint(1, 5),
        "target": random.randint(6, 10),
        "data": generate_large_data(5000)  # 5000-character transaction
    }
    
    response = requests.post(f"{API_BASE}/addTransaction", json=payload)
    
    if response.status_code == 202:
        print(f"✅ Transaction Sent: {response.json()['transactionID']}")
    else:
        print(f"❌ Error: {response.text}")

# Run 10 parallel transactions
threads = []
for _ in range(10):
    thread = threading.Thread(target=send_transaction)
    thread.start()
    threads.append(thread)

# Wait for all transactions to complete
for thread in threads:
    thread.join()
