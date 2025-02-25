'use client';

import React, { useState, useEffect } from "react";
import ReactFlow, { MiniMap, Controls, Background } from 'reactflow';
import 'reactflow/dist/style.css';

const API_BASE = "http://localhost:8080"; // Ensure this matches your backend

export default function BlockchainSharding() {
  const [nodes, setNodes] = useState([]);
  const [edges, setEdges] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchBlockchain();
  }, []);

  const fetchBlockchain = async () => {
    setLoading(true);
    setError("");

    try {
      const response = await fetch(`${API_BASE}/blockchain`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
      const data = await response.json();
      formatGraphData(data);
    } catch (err) {
      console.error("Error fetching blockchain data:", err);
      setError("Failed to fetch blockchain data. Ensure backend is running.");
    } finally {
      setLoading(false);
    }
  };

  const formatGraphData = (blocks) => {
    let newNodes = [];
    let newEdges = [];

    const shardColors = ["#FF5733", "#33FF57", "#3385FF", "#FF33A8", "#FFD700"];

    blocks.forEach((block, index) => {
      newNodes.push({
        id: block.index.toString(),
        data: {
          label: `Block ${block.index}`,
          hash: block.hash.slice(0, 6),
        },
        position: { x: block.index * 100, y: block.shard_id * 150 },
        style: {
          background: shardColors[block.shard_id % shardColors.length],
          color: "#fff",
          padding: 10,
          borderRadius: 10,
        },
      });

      if (block.index > 0) {
        newEdges.push({
          id: `e${block.index}`,
          source: (block.index - 1).toString(),
          target: block.index.toString(),
          animated: true,
          style: { strokeWidth: 2, stroke: "#ccc" },
        });
      }
    });

    setNodes(newNodes);
    setEdges(newEdges);
  };

  return (
    <div style={{ width: "100%", height: "80vh" }}>
      <h1 className="text-3xl font-bold text-center mb-6">
        Blockchain Sharding Visualisation
      </h1>

      <button
        onClick={fetchBlockchain}
        className="bg-blue-500 text-white p-2 rounded mb-4"
      >
        {loading ? "Loading..." : "Refresh"}
      </button>

      {error && <p className="text-red-500 text-center">{error}</p>}

      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        panOnScroll
        zoomOnScroll
        elementsSelectable
      >
        <MiniMap nodeStrokeWidth={3} />
        <Controls />
        <Background color="#aaa" gap={16} />
      </ReactFlow>
    </div>
  );
}
