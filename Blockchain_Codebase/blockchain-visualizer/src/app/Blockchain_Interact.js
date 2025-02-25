'use client';

import React, { useState, useEffect } from "react";
import ReactFlow, { MiniMap, Controls, Background } from "reactflow";
import "reactflow/dist/style.css";

const API_BASE = "http://localhost:8080"; // Ensure this matches your backend

export default function SpiderWebView() {
  const [nodes, setNodes] = useState([]);
  const [edges, setEdges] = useState([]);
  const [selectedBlock, setSelectedBlock] = useState(null);
  const [currentShard, setCurrentShard] = useState(null);

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

    // Organize blocks by shard
    let shardMap = {};
    blocks.forEach((block) => {
      if (!shardMap[block.shard_id]) {
        shardMap[block.shard_id] = [];
      }
      shardMap[block.shard_id].push(block);
    });

    let shardSpacing = 400; // Distance between shards
    let centerX = 400; // X coordinate for centering
    let centerY = 300; // Y coordinate for centering

    Object.entries(shardMap).forEach(([shardId, shardBlocks], index) => {
      let angleStep = (2 * Math.PI) / shardBlocks.length;
      let radius = 150 + shardBlocks.length * 10; // Distance from the center

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
    setSelectedBlock(node.data);
  };

  return (
    <div className="w-screen h-screen flex flex-col items-center bg-black text-white">
      <h1 className="text-3xl font-bold text-center mt-4">Spider-Web Blockchain View</h1>

      <button onClick={fetchBlockchain} className="mt-2 px-4 py-2 bg-blue-600 text-white rounded">
        Refresh
      </button>

      {/* Fullscreen Spider-Web Visualization */}
      <div className="w-full h-full relative">
        <ReactFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick}>
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>

        {/* Floating Block Details Box - Top Left */}
        {selectedBlock && (
          <div className="absolute top-4 left-4 bg-gray-800 text-white p-4 rounded shadow-lg w-72">
            <h2 className="text-lg font-bold">Block #{selectedBlock.index}</h2>
            <p><strong>Hash:</strong> {selectedBlock.hash.slice(0, 10)}...</p>
            <p><strong>Shard ID:</strong> {selectedBlock.shard_id}</p>
            <p><strong>Previous Hash:</strong> {selectedBlock.previous_hash.slice(0, 10)}...</p>
            <p><strong>Transactions:</strong> {selectedBlock.transactions.length}</p>
          </div>
        )}
      </div>
    </div>
  );
}
