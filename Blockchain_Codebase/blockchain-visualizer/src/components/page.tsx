"use client";

import { useEffect, useState } from "react";
import TransactionMetrics from "@/components/TransactionMetrics";

export default function TransactionsPage() {
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    fetch("http://localhost:8080/transactions")
      .then((res) => res.json())
      .then((data) => setLogs(data))
      .catch((err) => console.error("Failed to load logs", err));
  }, []);


  return (
   <div className="min-h-screen w-full bg-black text-white overflow-y-auto p-4">
      <TransactionMetrics logs={logs} />
    </div>
    
  );
  
}
