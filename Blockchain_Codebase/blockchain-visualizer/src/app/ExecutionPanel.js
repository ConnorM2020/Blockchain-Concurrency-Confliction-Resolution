'use client';


import React, { useState, useEffect } from "react";

const ExecutionPanel = ({
  sourceNode,
  targetNode = [],
  transactionData,
  setTransactionData,
  sendTransaction,
  sendParallelTransactions,
  sendCrossShardTransaction,
}) => {
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [logsOpen, setLogsOpen] = useState(false);
  const [transactionType, setTransactionType] = useState("all");
  const [transactionLogs, setTransactionLogs] = useState([]); 

  // Detect if this is a cross-shard transaction
  const isCrossShard =
    sourceNode &&
    targetNode.length > 0 &&
    targetNode.some((target) => target?.data?.shard_id !== sourceNode?.data?.shard_id);

  // Function to execute a transaction and fetch logs
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
        setTimeout(() => fetchTransactionLogs(type), 2000);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error executing transaction:", error);
        setLoading(false);
      });
  };

  const handleCrossShardTransaction = async () => {
    console.log("Executing Cross-Shard Transaction");

    if (!sourceNode || !targetNode.length) {
        alert("Please select a source and target node.");
        return;
    }
    try {
        const response = await fetch("http://localhost:8080/crossShardTransaction", {  // Check this URL
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ source: sourceNode, target: targetNode }),
        });

        if (!response.ok) throw new Error(`Error: ${response.status}`);

        const data = await response.json();
        console.log("Cross-Shard Transaction Response:", data);
        alert(data.message || "Cross-Shard Transaction Completed");
    } catch (error) {
        console.error("Error executing cross-shard transaction:", error);
        alert("Failed to execute cross-shard transaction. Please try again.");
    }
};

  // Fetch logs and filter based on selected transaction type
  const fetchTransactionLogs = async (type) => {
    console.log(`Fetching logs for: ${type}`); // Debugging output
    try {
      const response = await fetch("http://localhost:8080/transactionLogs");
      const data = await response.json();

      if (!data.logs || !Array.isArray(data.logs)) {
        console.error("Error: Invalid API response", data);
        setTransactionLogs([]);
        return;
      }

      // Filter logs based on transaction type
      const filteredLogs = type === "all"
        ? data.logs
        : data.logs.filter((log) => log.type.toLowerCase() === type);

       setTransactionLogs(type === "all" ? data.logs : data.logs.filter((log) => log.type.toLowerCase() === type));
    } catch (error) {
      console.error("Error fetching logs:", error);
    }
  };

  return (
    <div className="execution-panel text-center mb-6">
      <h2 className="text-xl font-bold text-white mb-4">🚀 Blockchain Execution Options</h2>

      {/* Execution Buttons */}
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
        <button
              onClick={handleCrossShardTransaction} 
              className="px-4 py-2 bg-yellow-500 hover:bg-yellow-600 text-white rounded-lg font-semibold"
          > Send Cross-Shard Transaction
          </button>
      </div>

      {/* Transaction Control Panel (Appears when Source & Targets Selected) */}
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
            <button onClick={sendTransaction} className="px-4 py-2 bg-purple-600 text-white rounded">
              Send Non-Parallel
            </button>

            <button onClick={sendParallelTransactions} className="px-4 py-2 bg-green-600 text-white rounded">
              Send Parallel
            </button>
            {/* Cross-Shard Transaction Button */}
            {isCrossShard && (
              <button onClick={sendCrossShardTransaction} className="px-4 py-2 bg-yellow-500 text-black rounded">
                Send Cross-Shard
              </button>
            )}
          </div>
          <button onClick={() => setTransactionData("")} className="w-full px-4 py-2 bg-gray-500 text-white rounded mt-4">
            Reset Selection
          </button>
        </div>
      )}

      {/* Transaction Logs Dropdown */}
      <div className="mt-6">
        <button
          onClick={() => setLogsOpen(!logsOpen)}
          className="w-full px-4 py-2 bg-gray-700 text-white rounded-lg"
        >
          {logsOpen ? "▼ Hide Recent Transaction Logs" : "▲ Show Recent Transaction Logs"}
        </button>

        {/* Log Content */}
        <div className={`overflow-hidden transition-all duration-500 ${logsOpen ? "max-h-96" : "max-h-0"}`}>
          <div className="bg-gray-900 p-4 rounded-lg shadow-lg text-white overflow-y-auto max-h-96">
            <h3 className="text-lg font-bold mb-2">📜 Transaction Log</h3>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700">
                  <th className="p-2">Tx ID</th>
                  <th className="p-2">Source → Target</th>
                  <th className="p-2">Type</th>
                  <th className="p-2">Exec Time (ms)</th>
                  <th className="p-2">Timestamp</th>
                </tr>
              </thead>
              <tbody>
                {transactionLogs.length > 0 ? (
                  transactionLogs.map((log, index) => (
                    <tr key={index} className="border-b border-gray-700">
                      <td className="p-2">{log.txID}</td>
                      <td className="p-2">{log.source} → {log.target}</td>
                      <td className={`p-2 ${log.type === "Sharded" ? "text-blue-400" : "text-red-400"}`}>
                        {log.type}
                      </td>
                      <td className="p-2">{log.execTime ? log.execTime.toFixed(3) : "N/A"}</td>
                      <td className="p-2">{new Date(log.timestamp).toLocaleString()}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan="5" className="p-2 text-center text-gray-400">No transactions found</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ExecutionPanel;