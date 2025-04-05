'use client';
import React from "react";
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  BarChart, Bar, PieChart, Pie, Cell, Legend
} from "recharts";

const COLORS = ["#4ade80", "#f87171"];

const TransactionMetrics = ({ logs }) => {
  const [showCombinedTPS, setShowCombinedTPS] = React.useState(false);
  if (!logs || logs.length === 0) return null;

  // Format data for charts
  const data = logs.map((log, index) => ({
    name: `Tx ${index + 1}`,
    execTime: log.execTime ?? 0,
    finalityTime: log.finalityTime ?? 0,
    propagationLatency: log.propagationLatency ?? 0, // keep in milliseconds

    type: log.type,
    tps: log.tps ?? 0,
    timestamp: new Date(log.timestamp).toLocaleTimeString(),
  }));

  const combinedTPSData = logs.map((log, index) => ({
    name: `Tx ${index + 1}`,
    shardedTPS: log.type?.toLowerCase() === "sharded" ? log.tps : 0,
    nonShardedTPS: log.type?.toLowerCase() === "non-sharded" ? log.tps : 0,
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

  // Sharded and Non-sharded TPS - differentiate betweent the two - making it more obvious 
  // Split TPS by type
  const tpsStats = logs.reduce(
    (acc, log) => {
      const type = log.type?.toLowerCase();
      if (type === "sharded") {
        acc.sharded.total += log.tps || 0;
        acc.sharded.count += 1;
      } else if (type === "non-sharded") {
        acc.nonSharded.total += log.tps || 0;
        acc.nonSharded.count += 1;
      }
      return acc;
    },
    {
      sharded: { total: 0, count: 0 },
      nonSharded: { total: 0, count: 0 },
    }
  );

  const avgShardedTPS =
    tpsStats.sharded.count > 0
      ? (tpsStats.sharded.total / tpsStats.sharded.count).toFixed(2)
      : "N/A";

  const avgNonShardedTPS =
    tpsStats.nonSharded.count > 0
      ? (tpsStats.nonSharded.total / tpsStats.nonSharded.count).toFixed(2)
      : "N/A";

    // Grouped TPS datasets
    const shardedTPS = logs
    .filter((log) => log.type?.toLowerCase() === "sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, tps: log.tps ?? 0 }));

    const nonShardedTPS = logs
    .filter((log) => log.type?.toLowerCase() === "non-sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, tps: log.tps ?? 0 }));

    
  // Compute average execution times
  const avgExec = data.reduce(
    (acc, log) => {
      const type = log.type.toLowerCase();
      acc[type].total += log.execTime;
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

    {/* Multi-Line Chart for all timings + TPS */}
    <div className="mb-6">
        <h4 className="text-lg font-semibold mb-2">Transaction Timings Comparison</h4>
        <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data}>
          <XAxis dataKey="name" />
          <YAxis yAxisId="left" />
          <YAxis
            yAxisId="right" orientation="right" domain={[0, 'auto']} tickFormatter={(val) => `${val} ms`}/>

          <Tooltip formatter={(value, name) => {
          let unit = "ms";
          if (name === "TPS") unit = "tx/s";
          else if (name === "Propagation Latency") unit = "ms";

          return [`${value.toFixed(2)} ${unit}`, name];
        }} />
            <Legend />
            <Line yAxisId="left" type="monotone" dataKey="execTime" stroke="#60a5fa" name="Execution Time" strokeWidth={2} dot />
            <Line yAxisId="left" type="monotone" dataKey="finalityTime" stroke="#facc15" name="Finality Time" strokeWidth={2} dot />
            <Line yAxisId="right" type="monotone" dataKey="propagationLatency" stroke="#fb7185" name="Propagation Latency" strokeWidth={2} dot />

          </LineChart>
        </ResponsiveContainer>
      </div>
    {/* Average Execution Time Comparison */}
       <div className="mb-6 text-sm bg-gray-800 p-4 rounded">
        <h4 className="text-lg font-semibold mb-2">Average Execution Time</h4>
        <p>
          <span className="text-green-600">Sharded:</span> {avgSharded} ms
        </p>
        <p>
          <span className="text-red-600">Non-Sharded:</span> {avgNonSharded} ms
        </p>
      </div>
    {/* Averages Overview */}
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
      <div className="bg-gray-800 p-4 rounded">
        <h5 className="font-semibold text-sm text-yellow-400">Avg Finality Time</h5>
        <p className="text-lg">
          {logs.length > 0
            ? (logs.reduce((acc, tx) => acc + (tx.finalityTime || 0), 0) / logs.length).toFixed(2)
            : "N/A"}{" "}ms
        </p>
      </div>
      <div className="bg-gray-800 p-4 rounded">
        <h5 className="font-semibold text-sm text-pink-400">Avg Propagation Latency</h5>
        <p className="text-lg">
          {logs.length > 0
            ? (logs.reduce((acc, tx) => acc + (tx.propagationLatency || 0), 0) / logs.length).toFixed(2)
            : "N/A"}{" "} ms
        </p>

      </div>
      <div className="bg-gray-800 p-4 rounded">
        <h5 className="font-semibold text-sm text-purple-400">Avg TPS</h5>
        <p className="text-lg text-green-500">Sharded: {avgShardedTPS} tx/s</p>
        <p className="text-lg text-red-500">Non-Sharded: {avgNonShardedTPS} tx/s</p>
      </div>
    </div>
    {/* Toggleable TPS Section */}
    <div className="text-right mb-4">
    <button
      onClick={() => setShowCombinedTPS(prev => !prev)}
      className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded"
    >
      {showCombinedTPS ? "View Split TPS" : "Compare TPS (Sharded vs Non-Sharded)"}
    </button>
  </div>
  {showCombinedTPS ? (
  <div className="bg-gray-800 p-4 rounded mb-6">
    <h4 className="text-lg font-semibold mb-2 text-cyan-300">Sharded vs Non-Sharded TPS</h4>
    <ResponsiveContainer width="100%" height={250}>
      <LineChart data={combinedTPSData}>
        <XAxis dataKey="name" />
        <YAxis />
        <Tooltip formatter={(value) => `${value?.toFixed(2)} tx/s`} />
        <Legend />
        <Line type="monotone" dataKey="shardedTPS" stroke="#4ade80" strokeWidth={2} dot animationDuration={300} />
        <Line type="monotone" dataKey="nonShardedTPS" stroke="#f87171" strokeWidth={2} dot animationDuration={300} />
      </LineChart>
    </ResponsiveContainer>
  </div>
) : (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        {/* Sharded TPS */}
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-green-300">Sharded TPS</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={shardedTPS}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(value) => `${value.toFixed(2)} tx/s`} />
              <Line
                type="monotone"
                dataKey="tps"
                stroke="#4ade80"
                strokeWidth={2}
                dot
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Non-Sharded TPS */}
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-red-300">Non-Sharded TPS</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={nonShardedTPS}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(value) => `${value.toFixed(2)} tx/s`} />
              <Line
                type="monotone"
                dataKey="tps"
                stroke="#f87171"
                strokeWidth={2}
                dot
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    )}

    {/* Bar Chart by Type */}
    <div className="mb-6">
      <h4 className="text-lg font-semibold mb-2">Execution Time by Type</h4>
      <ResponsiveContainer width="100%" height={250}>
      <BarChart data={[
        {
          type: "Sharded",
          execTime: avgExec.sharded.total,
          finalityTime: logs.filter(tx => tx.type === "Sharded").reduce((a, b) => a + (b.finalityTime || 0), 0),
        },
        {
          type: "Non-Sharded",
          execTime: avgExec["non-sharded"].total,
          finalityTime: logs.filter(tx => tx.type === "Non-Sharded").reduce((a, b) => a + (b.finalityTime || 0), 0),
        },
      ]}>
        <XAxis dataKey="type" />
        <YAxis />
        <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
        <Legend />
        <Bar dataKey="execTime" stackId="a" fill="#60a5fa" name="Execution Time" />
        <Bar dataKey="finalityTime" stackId="a" fill="#facc15" name="Finality Time" />
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
