'use client';

import React, { useState, useEffect } from "react";
import ReactFlow, { MiniMap, Controls, Background, ReactFlowProvider, useEdgesState, useNodesState } from "reactflow";
import "reactflow/dist/style.css";

const API_BASE = "http://localhost:8080";

export default function SpiderWebView() {
  const [nodes, setNodes] = useNodesState([]);
  const [edges, setEdges] = useEdgesState([]);
  const [selectedNodes, setSelectedNodes] = useState([]);
  const [selectedShard, setSelectedShard] = useState(1);
  const [shardOptions, setShardOptions] = useState([1, 2, 3, 4, 5]);

  // Transaction handling
  const [transactionModalOpen, setTransactionModalOpen] = useState(false);
  const [transactionData, setTransactionData] = useState("");
  const [sourceNode, setSourceNode] = useState(null);
  const [targetNode, setTargetNode] = useState(null);
  const [transactionStatus, setTransactionStatus] = useState(null);

  // Shard selection modal
  const [shardModalOpen, setShardModalOpen] = useState(false);

  useEffect(() => {
    fetchBlockchain();
  }, []);

  const fetchBlockchain = async () => {
    try {
      const response = await fetch(`${API_BASE}/blockchain`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
      const data = await response.json();
      formatSpiderWebData(data);
    } catch (err) {
      console.error("Error fetching blockchain data:", err);
    }
  };

  const formatSpiderWebData = (blocks) => {
    let newNodes = [];
    let newEdges = [];

    let shardMap = {};
    blocks.forEach((block) => {
      if (!shardMap[block.shard_id]) {
        shardMap[block.shard_id] = [];
      }
      shardMap[block.shard_id].push(block);
    });

    let shardSpacing = 400;
    let centerX = 400;
    let centerY = 300;

    let shardColors = [
      "#FF5733", "#33FF57", "#3385FF", "#FF33A1", "#FFAA33", "#AA33FF",
      "#33FFAA", "#FF3333", "#33A1FF", "#A1FF33", "#FF33FF", "#FFA133"
    ];

    Object.entries(shardMap).forEach(([shardId, shardBlocks], index) => {
      let angleStep = (2 * Math.PI) / shardBlocks.length;
      let radius = 150 + shardBlocks.length * 10;

      shardBlocks.forEach((block, i) => {
        let x = centerX + Math.cos(angleStep * i) * radius + index * shardSpacing;
        let y = centerY + Math.sin(angleStep * i) * radius;

        let shardIndex = parseInt(block.shard_id) || 0;
        let assignedColor = shardColors[shardIndex % shardColors.length];

        newNodes.push({
          id: block.index.toString(),
          data: { label: `Block ${block.index}`, ...block },
          position: { x, y },
          style: {
            background: assignedColor,
            color: "#fff",
            borderRadius: "5px",
            padding: "10px",
            cursor: "pointer",
            border: selectedNodes.includes(block.index) ? "3px solid yellow" : "none",
          },
          draggable: true,
        });

        if (block.previous_hash !== "0") {
          newEdges.push({
            id: `e${block.index}`,
            source: (block.index - 1).toString(),
            target: block.index.toString(),
            animated: true,
            style: { stroke: "#999" },
          });
        }
      });
    });

    setNodes(newNodes);
    setEdges(newEdges);
  };

  const handleNodeClick = (event, node) => {
    if (!sourceNode) {
      setSourceNode(node);
    } else if (!targetNode && node.id !== sourceNode.id) {
      setTargetNode(node);
      setTransactionModalOpen(true);
    }
  };

  const handleShardSelection = (event) => {
    setSelectedShard(Number(event.target.value));
  };

  const toggleNodeSelection = (nodeId) => {
    setSelectedNodes((prev) =>
      prev.includes(nodeId) ? prev.filter((id) => id !== nodeId) : [...prev, nodeId]
    );
  };
  const confirmShardCreation = async () => {
    if (selectedNodes.length === 0) {
      alert("Please select nodes to create a new shard.");
      return;
    }
  
    try {
      const response = await fetch(`${API_BASE}/createShard`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ shard_id: selectedShard, nodes: selectedNodes.map(Number) }),
      });
  
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      console.log(`✅ Shard ${selectedShard} created with nodes: ${selectedNodes}`);
      fetchBlockchain(); // ✅ Refresh the network to reflect the new sharded structure
      setShardModalOpen(false); // ✅ Close modal
      setSelectedNodes([]); // ✅ Reset selection
    } catch (err) {
      console.error("❌ Error creating shard:", err);
    }
  };
  

  const sendTransaction = async () => {
    if (!sourceNode || !targetNode || !transactionData.trim()) {
      alert("Please enter valid transaction data.");
      return;
    }

    const transactionPayload = {
      source: Number(sourceNode.id),
      target: Number(targetNode.id),
      data: transactionData,
    };

    try {
      const response = await fetch(`${API_BASE}/addTransaction`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(transactionPayload),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(`HTTP Error: ${response.status}, ${errorMessage}`);
      }

      setTransactionStatus("Transaction successfully sent! ✅");
      setTimeout(() => setTransactionStatus(null), 3000);
      setTransactionModalOpen(false);
      setSourceNode(null);
      setTargetNode(null);
      fetchBlockchain();
    } catch (error) {
      setTransactionStatus(`❌ Error sending transaction: ${error.message}`);
    }
  };

  return (
    <ReactFlowProvider>
      <div className="w-screen h-screen flex flex-col items-center bg-black text-white">
        <h1 className="text-3xl font-bold text-center mt-4">Spider-Web Blockchain View</h1>

        <div className="flex space-x-4 mt-2">
          <button onClick={fetchBlockchain} className="px-4 py-2 bg-blue-600 text-white rounded">
            Refresh
          </button>

          <button onClick={() => setShardModalOpen(true)} className="px-4 py-2 bg-purple-600 text-white rounded">
            Create New Shard
          </button>

          <button className="px-4 py-2 bg-red-600 text-white rounded" onClick={fetchBlockchain}>
            Reset Blockchain
          </button>
        </div>

        <div className="w-full h-full relative mt-4">
          <ReactFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick}>
            <MiniMap />
            <Controls />
            <Background />
          </ReactFlow>
        </div>

        {/* Transaction Modal */}
        {transactionModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Send Transaction</h2>
            <textarea
              className="w-full h-20 p-2 mt-3 bg-gray-700 text-white rounded"
              placeholder="Enter transaction details..."
              value={transactionData}
              onChange={(e) => setTransactionData(e.target.value)}
            />
            <button onClick={sendTransaction} className="px-4 py-2 bg-green-600 text-white rounded mt-4">
              Send
            </button>
          </div>
        )}

        {/* Shard Selection Modal */}
        {shardModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Select Nodes for Shard</h2>
            <div className="max-h-40 overflow-y-auto border p-2 rounded bg-gray-700">
              {nodes.map((node) => (
                <label key={node.id} className="flex items-center space-x-2 mb-2">
                  <input
                    type="checkbox"
                    checked={selectedNodes.includes(node.id)}
                    onChange={() => toggleNodeSelection(node.id)}
                  />
                  <span>{node.data.label}</span>
                </label>
              ))}
            </div>
            <button onClick={() => setShardModalOpen(false)} className="px-4 py-2 bg-red-600 text-white rounded mt-4">
              Close
            </button>

            <button onClick={confirmShardCreation} className="px-4 py-2 bg-green-600 text-white rounded">
              Allow
            </button>
            
          </div>
        )}
      </div>
    </ReactFlowProvider>
  );
}
