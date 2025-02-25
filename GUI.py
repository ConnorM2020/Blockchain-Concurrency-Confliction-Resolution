from flask import Flask, render_template, request, jsonify
import docker
import hashlib
import time

app = Flask(__name__)

docker_client = docker.from_env()

blockchain = []  # Blockchain data structure

class Block:
    def __init__(self, index, timestamp, container_id, previous_hash, hash):
        self.index = index
        self.timestamp = timestamp
        self.container_id = container_id
        self.previous_hash = previous_hash
        self.hash = hash

# Function to calculate the hash of a block
def calculate_hash(index, timestamp, container_id, previous_hash):
    record = f"{index}{timestamp}{container_id}{previous_hash}"
    return hashlib.sha256(record.encode()).hexdigest()

# Function to add a block to the blockchain
def add_block(container_id):
    previous_hash = blockchain[-1].hash if blockchain else "0"
    new_block = Block(
        index=len(blockchain),
        timestamp=time.strftime("%Y-%m-%d %H:%M:%S"),
        container_id=container_id,
        previous_hash=previous_hash,
        hash=calculate_hash(len(blockchain), time.strftime("%Y-%m-%d %H:%M:%S"), container_id, previous_hash)
    )
    blockchain.append(new_block)

@app.route('/')
def home():
    return render_template('index.html')

@app.route('/containers', methods=['GET'])
def get_containers():
    try:
        containers = docker_client.containers.list(all=True)
        container_data = [
            {
                "ID": container.id,
                "Image": container.image.tags[0] if container.image.tags else "<None>",
                "State": container.status,
                "Status": container.attrs['State']['Status']
            }
            for container in containers
        ]
        return jsonify(container_data)
    except Exception as e:
        return jsonify({"error": str(e)})

@app.route('/add_block', methods=['POST'])
def add_block_route():
    container_id = request.json.get('container_id')
    if not container_id:
        return jsonify({"error": "container_id is required"}), 400

    add_block(container_id)
    return jsonify({"message": "Block added to the blockchain successfully"})

@app.route('/blockchain', methods=['GET'])
def get_blockchain():
    blockchain_data = [
        {
            "Index": block.index,
            "Timestamp": block.timestamp,
            "ContainerID": block.container_id,
            "PreviousHash": block.previous_hash,
            "Hash": block.hash
        }
        for block in blockchain
    ]
    return jsonify(blockchain_data)

if __name__ == '__main__':
    app.run(debug=True)
