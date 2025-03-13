import React, { useState, useEffect } from "react";

const ExecutionPanel = () => {
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [transactionLogs, setTransactionLogs] = useState([]);

  // Fetch execution options from the backend on mount
  useEffect(() => {
    fetch("http://localhost:8080/executionOptions")
      .then((response) => response.json())
      .then((data) => setOptions(data.options))
      .catch((error) => console.error("Error fetching options:", error));

    // Fetch transaction logs on mount
    fetchTransactionLogs();
  }, []);

  // Function to execute a transaction and fetch logs after processing
  const handleExecute = (optionIndex) => {
    setLoading(true);
    fetch("http://localhost:8080/executeTransaction", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ option: optionIndex + 1 }),
    })
      .then((response) => response.json())
      .then((data) => {
        alert(data.message);

        // Introduce a short delay before fetching logs to ensure transactions are processed
        setTimeout(fetchTransactionLogs, 2000);

        setLoading(false);
      })
      .catch((error) => {
        console.error("Error executing transaction:", error);
        setLoading(false);
      });
  };

  // Fetch transaction logs from the backend
  const fetchTransactionLogs = () => {
    fetch("http://localhost:8080/transactionLogs")
      .then((response) => response.json())
      .then((data) => {
        console.log("Fetched logs:", data); // Debugging output
        if (!data.logs || !Array.isArray(data.logs)) {
          console.error("Error: Invalid API response", data);
          setTransactionLogs([]); // Prevent crash
          return;
        }
        setTransactionLogs(data.logs);
      })
      .catch((error) => console.error("Error fetching logs:", error));
  };

  return (
    <div className="execution-panel text-center mb-6">
      <h2 className="text-xl font-bold text-white mb-4">🚀 Blockchain Execution Options</h2>

      {/* Execution Buttons */}
      <div className="flex flex-wrap justify-center gap-4">
        {options.map((option, index) => (
          <button
            key={index}
            onClick={() => handleExecute(index)}
            disabled={loading}
            className={`px-4 py-2 rounded-lg font-semibold text-white transition ${
              index === 0 ? "bg-blue-500 hover:bg-blue-600" :
              index === 1 ? "bg-purple-500 hover:bg-purple-600" :
              index === 2 ? "bg-green-500 hover:bg-green-600" : ""
            }`}
          >
            {loading ? "Processing..." : option}
          </button>
        ))}
      </div>

      {/* Scrollable Transaction Log */}
      <div className="transaction-log mt-6 bg-gray-900 p-4 rounded-lg shadow-lg text-white max-h-96 overflow-y-auto">
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
  );
};

export default ExecutionPanel;
