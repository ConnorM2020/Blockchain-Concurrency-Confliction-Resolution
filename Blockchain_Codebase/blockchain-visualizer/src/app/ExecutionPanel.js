'use client';

import React, { useState, useEffect } from "react";

const ExecutionPanel = () => {
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [transactionLogs, setTransactionLogs] = useState([]);
  const [logsOpen, setLogsOpen] = useState(false); // Toggle state for logs panel
  const [transactionType, setTransactionType] = useState("all"); // Default to showing all transactions

  // Fetch execution options from the backend on mount
  useEffect(() => {
    fetch("http://localhost:8080/executionOptions")
      .then((response) => response.json())
      .then((data) => setOptions(data.options))
      .catch((error) => console.error("Error fetching options:", error));

    fetchTransactionLogs("all"); // Fetch all transactions initially
  }, []);

  // Function to execute a transaction and fetch logs
  const handleExecute = (type) => {
    setLoading(true);
    setTransactionType(type); // Update type immediately for UI
    console.log(`Executing ${type} transaction`); // Debugging output

    fetch("http://localhost:8080/executeTransaction", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ option: type === "sharded" ? 1 : 2 }),
    })
    .then((response) => response.json())
    .then((data) => {
      console.log("Server response:", data); // Debugging output
      alert(data.message);
      setTimeout(() => fetchTransactionLogs(type), 2000); // Pass type explicitly
      setLoading(false);
    })
    .catch((error) => {
      console.error("Error executing transaction:", error);
      setLoading(false);
    });
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

      setTransactionLogs(filteredLogs);
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
      </div>

      {/* Transaction Logs Dropdown */}
      <div className="mt-6">
        <button
          onClick={() => setLogsOpen(!logsOpen)}
          className="w-full px-4 py-2 bg-gray-700 text-white rounded-lg"
        >
          {logsOpen ? "▼ Hide Transaction Logs" : "▲ Show Transaction Logs"}
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
