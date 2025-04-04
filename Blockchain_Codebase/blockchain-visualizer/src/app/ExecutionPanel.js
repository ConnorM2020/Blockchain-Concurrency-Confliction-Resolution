'use client';

import React, { useState, useEffect } from "react";

const ExecutionPanel = ({
  sourceNode,
  targetNode = [],
  setSourceNode,
  setTargetNode,
  setNodes, 
  transactionData,
  setTransactionData,
  sendTransaction,
  sendParallelTransactions,
}) => {
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);

  const [transactionType, setTransactionType] = useState("all");


  const handleExecute = (type) => {
    setLoading(true);
    setTransactionType(type);
    console.log(`Executing ${type} transaction`);

    fetch("http://localhost:8080/executeTransaction", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ option: type === "sharded" ? 1 : 2 }),
    })
      .then((response) => response.json())
      .then((data) => {
        console.log("Server response:", data);
        alert(data.message);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error executing transaction:", error);
        setLoading(false);
      });
  };

  return (
    <div className="execution-panel text-center mb-6">
      <h2 className="text-xl font-bold text-white mb-4">🚀 Blockchain Execution Options</h2>

      <div className="flex flex-wrap justify-center gap-4">
        <button
          onClick={() => handleExecute("sharded")}
          disabled={loading}
          className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-semibold"
        >
          {loading ? "Processing..." : "Run Sharded Transactions"}
        </button>

        <button
          onClick={() => handleExecute("non-sharded")}
          disabled={loading}
          className="px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white rounded-lg font-semibold"
        >
          {loading ? "Processing..." : "Run Non-Sharded Transactions"}
        </button>
      </div>

      {sourceNode && targetNode.length > 0 && (
        <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
          <h2 className="text-xl font-bold mb-4 text-white">Transaction Options</h2>
          <p className="text-white mb-2">
            <strong>Source Node:</strong> {sourceNode?.data?.label}
          </p>
          <p className="text-white">
            <strong>Target Nodes:</strong> {targetNode.map((t) => t.data.label).join(", ")}
          </p>

          <textarea
            className="w-full h-20 p-2 bg-gray-700 text-white rounded mt-4"
            value={transactionData}
            onChange={(e) => setTransactionData(e.target.value)}
            placeholder="Enter transaction details..."
          />
          <div className="flex flex-wrap justify-between mt-4">
            <button
              onClick={() => sendTransaction("non-sharded")}
              className="px-4 py-2 bg-purple-600 text-white rounded"
            > Send Non-Sharded Transaction
            </button>
            <button
              onClick={() => sendParallelTransactions()}
              className="px-4 py-2 bg-green-600 text-white rounded"
            > Send Sharded Transaction
            </button>
          </div>
          <button
            onClick={() => {
              setSourceNode(null);
              setTargetNode([]);
              setTransactionData("");
              setNodes((prevNodes) =>
                prevNodes.map((node) => {
                  if (node.type === "shardBubble") return node;
                  return {
                    ...node,
                    style: {
                      ...node.style,
                      background: node.data?.color || "#444",
                      color: "#fff",
                    },
                  };
                })
              );
            }}
            className="w-full px-4 py-2 bg-red-600 text-white rounded mt-2"
          > Back
          </button>
        </div>
      )}
    
    </div>
  );
};

export default ExecutionPanel;
