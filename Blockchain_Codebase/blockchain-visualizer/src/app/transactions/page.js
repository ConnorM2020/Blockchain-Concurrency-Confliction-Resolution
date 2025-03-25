"use client"; // Required for client-side fetching

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation"; // Correct router import for App Router
import TransactionMetrics from "@/app/TransactionMetrics";

const TransactionsPage = () => {
  const router = useRouter();
  const [transactions, setTransactions] = useState([]);
  const [sortConfig, setSortConfig] = useState({ key: null, direction: "asc" });

  // Fetch transactions from the backend
  useEffect(() => {
    fetch("http://localhost:8080/allTransactions")
      .then((res) => res.json())
      .then((data) => setTransactions(data.logs))
      .catch((err) => console.error("Error fetching transactions:", err));
      
  }, []);
  // Sorting function
  const sortTransactions = (key) => {
    let direction = "asc";
    if (sortConfig.key === key && sortConfig.direction === "asc") {
      direction = "desc";
    }

    const sortedData = [...transactions].sort((a, b) => {
      if (key === "execTime") {
        return direction === "asc" ? a.execTime - b.execTime : b.execTime - a.execTime;
      } else if (key === "type") {
        return direction === "asc" ? a.type.localeCompare(b.type) : b.type.localeCompare(a.type);
      } else if (key === "sourceTarget") {
        const aVal = `${a.source} → ${a.target}`;
        const bVal = `${b.source} → ${b.target}`;
        return direction === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      }
      return 0;
    });

    setTransactions(sortedData);
    setSortConfig({ key, direction });
  };

  return (
    <div className="p-6 bg-black text-white min-h-screen">
      <h1 className="text-2xl font-bold mb-4">📜 All Transactions</h1>

      {/* Back to Dashboard Button */}
      <button
        onClick={() => router.push("/")}
        className="mb-4 px-4 py-2 bg-blue-600 text-white rounded"
      >
        ← Back to Dashboard
      </button>

      {/* Transaction Table */}
      <div className="overflow-x-auto">
        <table className="w-full border-collapse border border-gray-700 text-sm">
          <thead>
            <tr className="border-b border-gray-700 bg-gray-800">
              <th className="p-2">Tx ID</th>
              <th
                className="p-2 cursor-pointer hover:text-blue-400"
                onClick={() => sortTransactions("sourceTarget")}
              >
                Source → Target {sortConfig.key === "sourceTarget" && (sortConfig.direction === "asc" ? "↑" : "↓")}
              </th>
              <th
                className="p-2 cursor-pointer hover:text-blue-400"
                onClick={() => sortTransactions("type")}
              >
                Type {sortConfig.key === "type" && (sortConfig.direction === "asc" ? "↑" : "↓")}
              </th>
              <th
                className="p-2 cursor-pointer hover:text-blue-400"
                onClick={() => sortTransactions("execTime")}
              >
                Exec Time (ms) {sortConfig.key === "execTime" && (sortConfig.direction === "asc" ? "↑" : "↓")}
              </th>
              <th className="p-2">Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {transactions.length > 0 ? (
              transactions.map((tx, index) => (
                <tr key={index} className="border-b border-gray-700">
                  <td className="p-2">{tx.txID}</td>
                  <td className="p-2">{tx.source} → {tx.target}</td>
                  <td className={`p-2 ${tx.type === "Sharded" ? "text-blue-400" : "text-red-400"}`}>
                    {tx.type}
                  </td>
                  <td className="p-2">{tx.execTime ? tx.execTime.toFixed(3) : "N/A"}</td>
                  <td className="p-2">{new Date(tx.timestamp).toLocaleString()}</td>
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
        {/* 📊 Metrics Section */}
        {transactions.length > 0 && (
        <div className="mt-10">
          <TransactionMetrics logs={transactions} />
        </div>
      )}
    </div>
  );
};

export default TransactionsPage;
      