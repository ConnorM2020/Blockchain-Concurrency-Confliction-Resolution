'use client';

import React, { useState, useEffect } from "react";
import ReactFlow, { MiniMap, Controls, Background, useEdgesState, useNodesState } from "reactflow";
import "reactflow/dist/style.css";
import TransactionModel from "./TransactionModel";

const API_BASE = "http://localhost:8080"; // Ensure this matches your backend

export default function SpiderWebView() {
  const [nodes, setNodes] = useNodesState([]);
  const [edges, setEdges] = useEdgesState([]);
  const [selectedBlock, setSelectedBlock] = useState(null);
  const [currentShard, setCurrentShard] = useState(null);
  const [firstSelectedNode, setFirstSelectedNode] = useState(null); // Track selected nodes
  const [isModelOpen, setModelOpen] = useState(false);

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

    Object.entries(shardMap).forEach(([shardId, shardBlocks], index) => {
      let angleStep = (2 * Math.PI) / shardBlocks.length;
      let radius = 150 + shardBlocks.length * 10;

      shardBlocks.forEach((block, i) => {
        let x = centerX + Math.cos(angleStep * i) * radius + index * shardSpacing;
        let y = centerY + Math.sin(angleStep * i) * radius;

        newNodes.push({
          id: block.index.toString(),
          data: { label: `Block ${block.index}`, ...block },
          position: { x, y },
          style: {
            background: shardId % 2 === 0 ? "#FF5733" : "#33FF57",
            color: "#fff",
            borderRadius: "5px",
            padding: "10px",
            cursor: "pointer",
            border: firstSelectedNode?.id === block.index.toString() ? "2px solid yellow" : "none",
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
    if (!firstSelectedNode) {
      setFirstSelectedNode(node);
    } else {
      if (firstSelectedNode.id !== node.id) {
        // Create a transaction between two selected nodes
        const newEdge = {
          id: `e${firstSelectedNode.id}-${node.id}`,
          source: firstSelectedNode.id,
          target: node.id,
          animated: true,
          style: { stroke: "blue" },
        };

        setEdges((prevEdges) => [...prevEdges, newEdge]);
        sendTransactionToBackend(firstSelectedNode.id, node.id);

        // Reset selection after connection
        setFirstSelectedNode(null);
      }
    }
  };

  const sendTransactionToBackend = async (source, target, data) => {
    const transactionData = {
        source: Number(source),  
        target: Number(target),  
        data: data || "Default transaction data" 
    };
    console.log("📤 Sending Transaction Data:", JSON.stringify(transactionData));
    try {
        const response = await fetch(`${API_BASE}/addTransaction`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(transactionData) 
        });

        if (!response.ok) {
            const errorMessage = await response.text();
            throw new Error(`HTTP Error: ${response.status}, ${errorMessage}`);
        }

        console.log("✅ Transaction successfully sent.");
    } catch (error) {
        console.error("❌ Error sending transaction:", error);
    }
};


  return (
    <div className="w-screen h-screen flex flex-col items-center bg-black text-white">
      <h1 className="text-3xl font-bold text-center mt-4">Spider-Web Blockchain View</h1>

      <button onClick={fetchBlockchain} className="mt-2 px-4 py-2 bg-blue-600 text-white rounded">
        Refresh
      </button>

      <div className="w-full h-full relative">
        <ReactFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick}>
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>

        {selectedBlock && (
          <div className="absolute top-4 left-4 bg-gray-800 text-white p-4 rounded shadow-lg w-72">
            <h2 className="text-lg font-bold">Block #{selectedBlock.index}</h2>
            <p><strong>Hash:</strong> {selectedBlock.hash.slice(0, 10)}...</p>
            <p><strong>Shard ID:</strong> {selectedBlock.shard_id}</p>
            <p><strong>Previous Hash:</strong> {selectedBlock.previous_hash.slice(0, 10)}...</p>
            <p><strong>Transactions:</strong></p>
            {selectedBlock.transactions.length > 0 ? (
              <ul>
                {selectedBlock.transactions.map((tx, index) => (
                  <li key={index}>
                    <p><strong>Tx ID:</strong> {tx.transaction_id}</p>
                    <p><strong>Container ID:</strong> {tx.container_id}</p>
                    <p><strong>Timestamp:</strong> {tx.timestamp}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <p>No transactions recorded.</p>
            )}
          </div>
        )}
        {/* Transaction Modal */}
        <TransactionModel
          isOpen={isModelOpen}
          onClose={() => setModelOpen(false)}
          onSubmit={() => {}}
        />
      </div>
    </div>
  );
}
