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

  // Count transactions by type
  const typeCounts = logs.reduce(
    (acc, log) => {
      const type = log.type.toLowerCase();
      acc[type] = (acc[type] || 0) + 1;
      return acc;
    },
    { sharded: 0, "non-sharded": 0 }
  );

  // Pie chart data
  const pieData = [
    { name: "Sharded", value: typeCounts["sharded"] },
    { name: "Non-Sharded", value: typeCounts["non-sharded"] },
  ];

  // Compute average execution times
  const avgExec = logs.reduce(
    (acc, log) => {
      const type = log.type.toLowerCase();
      acc[type].total += log.execTime || 0;
      acc[type].count += 1;
      return acc;
    },
    {
      sharded: { total: 0, count: 0 },
      "non-sharded": { total: 0, count: 0 }
    }
  );
  const avgSharded = avgExec.sharded.count > 0
    ? (avgExec.sharded.total / avgExec.sharded.count).toFixed(3)
    : "N/A";
  const avgNonSharded = avgExec["non-sharded"].count > 0
    ? (avgExec["non-sharded"].total / avgExec["non-sharded"].count).toFixed(3)
    : "N/A";

  return (
    <div className="bg-gray-900 p-6 rounded-lg mt-6 shadow-lg text-white">

    {/* Average Execution Time Comparison */}
      <div className="mb-6 text-sm bg-gray-800 p-4 rounded">
        <h4 className="text-lg font-semibold mb-2">Average Execution Time</h4>
        <p>
          <span className="text-green-400">Sharded:</span> {avgSharded} ms
        </p>
        <p>
          <span className="text-red-400">Non-Sharded:</span> {avgNonSharded} ms
        </p>
      </div>

      {/* Why Use Sharding */}
      <div className="mt-4 text-sm bg-gray-800 p-4 rounded">
        <h4 className="text-lg font-semibold mb-2">Why Use Sharding?</h4>
        <ul className="list-disc list-inside space-y-1">
          <li><strong>Scalability:</strong> Transactions are processed in parallel across shards.</li>
          <li><strong>Performance:</strong> Lower average execution time when sharded logic is used.</li>
          <li><strong>Conflict Isolation:</strong> Sharding makes concurrency conflicts more localised and manageable.</li>
        </ul>
      </div>

      <h3 className="text-xl font-bold mb-4">Transaction Metrics</h3>

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

      {/* Pie Chart for Distribution */}
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
