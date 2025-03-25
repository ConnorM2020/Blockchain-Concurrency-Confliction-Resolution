'use client';
import React from "react";
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  BarChart, Bar, PieChart, Pie, Cell, Legend
} from "recharts";

const COLORS = ["#4ade80", "#f87171"];

const TransactionMetrics = ({ logs }) => {
  if (!logs || logs.length === 0) return null;

  // Format data for charts
  const data = logs.map((log, index) => ({
    name: `Tx ${index + 1}`,
    execTime: log.execTime,
    type: log.type,
    timestamp: new Date(log.timestamp).toLocaleTimeString(),
  }));

  const typeCounts = logs.reduce(
    (acc, log) => {
      const type = log.type.toLowerCase();
      acc[type] = (acc[type] || 0) + 1;
      return acc;
    },
    { sharded: 0, "non-sharded": 0 }
  );

  const pieData = [
    { name: "Sharded", value: typeCounts["sharded"] },
    { name: "Non-Sharded", value: typeCounts["non-sharded"] },
  ];

  return (
    <div className="bg-gray-900 p-6 rounded-lg mt-6 shadow-lg text-white">
      <h3 className="text-xl font-bold mb-4">📊 Transaction Metrics</h3>

      {/* Line Chart for Execution Time */}
      <div className="mb-6">
        <h4 className="text-lg font-semibold mb-2">Execution Time (ms)</h4>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={data}>
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Line type="monotone" dataKey="execTime" stroke="#60a5fa" strokeWidth={2} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Bar Chart by Type */}
      <div className="mb-6">
        <h4 className="text-lg font-semibold mb-2">Execution Time by Type</h4>
        <ResponsiveContainer width="100%" height={250}>
          <BarChart data={data}>
            <XAxis dataKey="type" />
            <YAxis />
            <Tooltip />
            <Bar dataKey="execTime" fill="#34d399" />
          </BarChart>
        </ResponsiveContainer>
      </div>

      {/* Pie Chart for Type Distribution */}
      <div className="mb-6">
        <h4 className="text-lg font-semibold mb-2">Transaction Type Distribution</h4>
        <ResponsiveContainer width="100%" height={250}>
          <PieChart>
            <Pie
              data={pieData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              outerRadius={80}
              label
            >
              {pieData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
              ))}
            </Pie>
            <Legend />
            <Tooltip />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default TransactionMetrics;
