import React, { useState } from "react";

export default function TransactionModel({ selectedBlock, onClose, onSubmit }) {
  const [transactions, setTransactions] = useState([{ id: 1, data: "" }]);

  const addTransaction = () => {
    setTransactions([...transactions, { id: transactions.length + 1, data: "" }]);
  };

  const updateTransaction = (id, value) => {
    setTransactions(
      transactions.map((t) => (t.id === id ? { ...t, data: value } : t))
    );
  };

  const submitTransactions = () => {
    onSubmit(transactions.map((t) => t.data));
  };

  if (!selectedBlock) return null;

  return (
    <div className="fixed top-0 left-0 w-full h-full flex items-center justify-center bg-black bg-opacity-50">
      <div className="bg-white text-black p-6 rounded-lg shadow-lg w-96">
        <h2 className="text-lg font-bold mb-2">Add Transactions to Block #{selectedBlock.index}</h2>

        {transactions.map((t) => (
          <input
            key={t.id}
            type="text"
            placeholder="Enter transaction data"
            className="border p-2 w-full mb-2"
            value={t.data}
            onChange={(e) => updateTransaction(t.id, e.target.value)}
          />
        ))}

        <div className="flex justify-between mt-4">
          <button onClick={addTransaction} className="bg-green-500 text-white px-4 py-2 rounded">
            + Add Transaction
          </button>

          <button onClick={submitTransactions} className="bg-blue-500 text-white px-4 py-2 rounded">
            Submit
          </button>

          <button onClick={onClose} className="bg-red-500 text-white px-4 py-2 rounded">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
