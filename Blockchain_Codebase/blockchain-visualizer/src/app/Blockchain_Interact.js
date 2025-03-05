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
  const [shardModalOpen, setShardModalOpen] = useState(false);
  const [pendingTransactions, setPendingTransactions]= useState([]);
  // Transaction handling
  const [transactionModalOpen, setTransactionModalOpen] = useState(false);
  const [transactionData, setTransactionData] = useState("");
  const [sourceNode, setSourceNode] = useState(null);
  const [targetNode, setTargetNode] = useState(null);
  const [transactionStatus, setTransactionStatus] = useState({})
  const [parallelModalOpen, setParallelModalOpen] = useState(false)
  const [parallelTransactions, setParallelTransactions] = useState([]);


  useEffect(() => {
    fetchBlockchain();
    if (pendingTransactions.length > 0) {
      const interval = setInterval(checkPendingTransactions, 2000);
      return () => clearInterval(interval);
    }
  }, [pendingTransactions]);  
  

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
  
      const result = await response.json();
      const transactionID = result.transactionID;
  
      // Track this transaction as pending
      setPendingTransactions((prev) => [...prev, transactionID]);
  
      setTransactionModalOpen(false);
    } catch (error) {
      console.error("❌ Error sending transaction:", error);
    }
  };
  const checkPendingTransactions = async () => {
    if (pendingTransactions.length === 0) return;
  
    const updatedPending = [];
    for (const transactionID of pendingTransactions) {
      try {
        const response = await fetch(`${API_BASE}/transactionStatus/${transactionID}`);
        if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
        const data = await response.json();
        if (data.status === "completed") {
          setTransactionStatus((prev) => ({ ...prev, [transactionID]: "✅ Completed" }));
          fetchBlockchain(); // Refresh UI when transaction is done
        } else {
          updatedPending.push(transactionID); // Keep tracking if not completed
        }
      } catch (err) {
        console.error(`❌ Error checking status for ${transactionID}:`, err);
      }
    }
    setPendingTransactions(updatedPending);
  };
  
  const checkTransactionStatus = async (transactionID) => {
    try {
      const response = await fetch(`${API_BASE}/transactionStatus/${transactionID}`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      const data = await response.json();
      return data.status;
    } catch (error) {
      console.error("Error checking transaction status:", error);
      return "Unknown";
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

    let shardColors = [
      "#FF5733", "#33FF57", "#3385FF", "#FF33A1", "#FFAA33", "#AA33FF",
      "#33FFAA", "#FF3333", "#33A1FF", "#A1FF33", "#FF33FF", "#FFA133"
    ];

    Object.entries(shardMap).forEach(([shardId, shardBlocks], index) => {
      let angleStep = (2 * Math.PI) / shardBlocks.length;
      let radius = 150 + shardBlocks.length * 10;

      shardBlocks.forEach((block, i) => {
        let x = 400 + Math.cos(angleStep * i) * radius + index * 400;
        let y = 300 + Math.sin(angleStep * i) * radius;

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
  const toggleNodeSelection = (nodeId) => {
    setSelectedNodes((prev) =>
      prev.includes(nodeId) ? prev.filter((id) => id !== nodeId) : [...prev, nodeId]
    );
  };

  const updateParallelTransaction = (index, field, value) => {
    setParallelTransactions((prev) => {
        const updatedTransactions = [...prev];
        updatedTransactions[index] = { ...updatedTransactions[index], [field]: value };
        return updatedTransactions;
    });
};
  const addNewParallelTransaction = () => {
    setParallelTransactions([...parallelTransactions, { source: "", target: "", data: "" }]);
  };

  const sendParallelTransactions = async () => {
    if (parallelTransactions.length === 0) {
        alert("Please enter at least one transaction.");
        return;
    }
    // Ensure transactions have valid source, target, and data fields
    const formattedTransactions = parallelTransactions.map((tx) => ({
        source: Number(tx.source),
        target: Number(tx.target),
        data: tx.data.trim(),
    })).filter(tx => tx.source && tx.target && tx.data); // Remove empty transactions

    if (formattedTransactions.length === 0) {
        alert("Please ensure all transactions have valid data.");
        return;
    }
    try {
        const response = await fetch(`${API_BASE}/addParallelTransactions`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(formattedTransactions),
        });

        if (!response.ok) {
            const errorMessage = await response.text();
            throw new Error(`HTTP Error: ${response.status}, ${errorMessage}`);
        }

        const result = await response.json();
        const transactionIDs = result.transactionIDs;

        setPendingTransactions((prev) => [...prev, ...transactionIDs]);
        setParallelModalOpen(false);
        setParallelTransactions([]); // Reset transactions input
    } catch (error) {
        console.error("❌ Error sending parallel transactions:", error);
    }
};

  const confirmShardCreation = async () => {
    if (selectedNodes.length === 0) {
      alert("Please select nodes to assign to a shard.");
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/assignNodesToShard`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ shard_id: selectedShard, nodes: selectedNodes.map(Number) }),
      });

      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);

      console.log(`✅ Nodes ${selectedNodes} assigned to Shard ${selectedShard}`);
      fetchBlockchain();  // Refresh the visualization
      setSelectedNodes([]); 
      setShardModalOpen(false);
    } catch (err) {
      console.error("❌ Error assigning nodes to shard:", err);
    }
  };

  const resetBlockchain = async () => {
    try {
      const response = await fetch(`${API_BASE}/resetBlockchain`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
  
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      console.log("✅ Blockchain reset successfully.");
      fetchBlockchain(); // Refresh UI after reset
    } catch (err) {
      console.error("❌ Error resetting blockchain:", err);
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
  
          <button onClick={() => setParallelModalOpen(true)} className="px-4 py-2 bg-green-600 text-white rounded">
            Parallel Transactions
          </button>
  
          <button className="px-4 py-2 bg-red-600 text-white rounded" onClick={resetBlockchain}>
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
  
        {/* Transaction Status Overlay */}
        {Object.keys(transactionStatus || {}).length > 0 &&
          Object.keys(transactionStatus).map((txID) => (
            <div key={txID} className="absolute top-4 right-4 bg-gray-800 text-white px-4 py-2 rounded">
              {txID}: {transactionStatus[txID]}
            </div>
          ))}
  
        {/* Transaction Modal */}
        {transactionModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Send Transaction</h2>
            <textarea
              className="w-full h-20 p-2 bg-gray-700 text-white rounded"
              value={transactionData}
              onChange={(e) => setTransactionData(e.target.value)}
            />
            <button onClick={sendTransaction} className="px-4 py-2 bg-green-600 text-white rounded mt-4">
              Send
            </button>
          </div>
        )}
  
        {/* Parallel Transactions Modal */}
        {parallelModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Send Parallel Transactions</h2>
  
            {parallelTransactions.map((tx, index) => (
              <div key={index} className="mb-2">
                <input
                  type="number"
                  placeholder="Source Node"
                  value={tx.source}
                  onChange={(e) => updateParallelTransaction(index, "source", e.target.value)}
                  className="w-full p-2 mb-2 bg-gray-700 text-white rounded"
                />
                <input
                  type="number"
                  placeholder="Target Node"
                  value={tx.target}
                  onChange={(e) => updateParallelTransaction(index, "target", e.target.value)}
                  className="w-full p-2 mb-2 bg-gray-700 text-white rounded"
                />
                <input
                  type="text"
                  placeholder="Transaction Data"
                  value={tx.data}
                  onChange={(e) => updateParallelTransaction(index, "data", e.target.value)}
                  className="w-full p-2 bg-gray-700 text-white rounded"
                />
              </div>
            ))}
  
            <button onClick={addNewParallelTransaction} className="px-4 py-2 bg-blue-500 text-white rounded mr-2">
              + Add Transaction
            </button>
            <button onClick={sendParallelTransactions} className="px-4 py-2 bg-green-500 text-white rounded">
              Send Transactions
            </button>
          </div>
        )}
  
        {/* Shard Selection Modal */}
        {shardModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Assign Nodes to a Shard</h2>
  
            {/* Dropdown for selecting Shard ID */}
            <label className="block text-white mb-2">Select Shard ID:</label>
            <select
              value={selectedShard}
              onChange={(e) => setSelectedShard(Number(e.target.value))}
              className="w-full p-2 mb-4 bg-gray-700 text-white rounded"
            >
              {shardOptions.map((shard) => (
                <option key={shard} value={shard}>
                  Shard {shard}
                </option>
              ))}
            </select>
  
            {/* Checkboxes for selecting nodes */}
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
  
            <button onClick={confirmShardCreation} className="px-4 py-2 bg-green-600 text-white rounded mt-4">
              Assign to Shard
            </button>
          </div>
        )}
      </div>
    </ReactFlowProvider>
  );
}  