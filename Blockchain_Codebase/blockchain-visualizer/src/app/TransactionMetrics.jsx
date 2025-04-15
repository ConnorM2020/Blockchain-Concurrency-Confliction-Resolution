'use client';
import React, { useState } from "react";
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  BarChart, Bar, PieChart, Pie, Cell, Legend, LabelList, ReferenceDot 
} from "recharts";


const COLORS = ["#4ade80", "#f87171"];

const TransactionMetrics = ({ logs }) => {
  const [showCombinedTPS, setShowCombinedTPS] = useState(false);
  const [showCombinedExec, setShowCombinedExec] = useState(false);
  const [showCombinedFinality, setShowCombinedFinality] = useState(false);
  const [showCombinedPropagation, setShowCombinedPropagation] = useState(false);

  if (!logs || logs.length === 0) return null;
  const tpsStats = logs.reduce((acc, log) => {
    const type = log.type?.toLowerCase();
    if (type === "sharded") {
      acc.sharded.total += log.tps || 0;
      acc.sharded.count++;
    } else if (type === "non-sharded") {
      acc.nonSharded.total += log.tps || 0;
      acc.nonSharded.count++;
    }
    return acc;
  }, {
    sharded: { total: 0, count: 0 },
    nonSharded: { total: 0, count: 0 }
  });


  const combinedExecData = logs.map((log, i) => ({
    name: `Tx ${i + 1}`,
    shardedExec: log.type?.toLowerCase() === "sharded" ? log.execTime : null,
    nonShardedExec: log.type?.toLowerCase() === "non-sharded" ? log.execTime : null,
  }));  
  
  const combinedFinalityData = logs.map((log, i) => ({
    name: `Tx ${i + 1}`,
    shardedFinality: log.type?.toLowerCase() === "sharded" ? log.finalityTime : null,
    nonShardedFinality: log.type?.toLowerCase() === "non-sharded" ? log.finalityTime : null,
  }));
  
  const combinedPropagationData = logs.map((log, i) => ({
    name: `Tx ${i + 1}`,
    shardedPropagation: log.type?.toLowerCase() === "sharded" ? log.propagationLatency : null,
    nonShardedPropagation: log.type?.toLowerCase() === "non-sharded" ? log.propagationLatency : null,
  }));

  const combinedTPSData = logs.map((log, index) => ({
    name: `Tx ${index + 1}`,
    shardedTPS: log.type?.toLowerCase() === "sharded" ? log.tps : 0,
    nonShardedTPS: log.type?.toLowerCase() === "non-sharded" ? log.tps : 0,
  }));
  
  const avgShardedTPS = tpsStats.sharded.count ? (tpsStats.sharded.total / tpsStats.sharded.count).toFixed(2) : "N/A";
  const avgNonShardedTPS = tpsStats.nonSharded.count ? (tpsStats.nonSharded.total / tpsStats.nonSharded.count).toFixed(2) : "N/A";


  const typeCounts = logs.reduce((acc, log) => {
    const type = log.type?.toLowerCase();
    if (type) acc[type] = (acc[type] || 0) + 1;
    return acc;
  }, { sharded: 0, "non-sharded": 0 });

  const pieData = [
    { name: "Sharded", value: typeCounts["sharded"] },
    { name: "Non-Sharded", value: typeCounts["non-sharded"] },
  ];
  
  const data = logs.map((log, index) => ({
    name: `Tx ${index + 1}`,
    execTime: log.execTime ?? 0,
    finalityTime: log.finalityTime ?? 0,
    propagationLatency: log.propagationLatency ?? 0,
    type: log.type,
    tps: log.tps ?? 0,
    timestamp: new Date(log.timestamp).toLocaleTimeString(),
  }));

  const shardedTPS = logs.filter(log => log.type?.toLowerCase() === "sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, tps: log.tps ?? 0 }));

  const nonShardedTPS = logs.filter(log => log.type?.toLowerCase() === "non-sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, tps: log.tps ?? 0 }));

  const avgExec = data.reduce((acc, log) => {
    const type = log.type?.toLowerCase();
    if (acc[type]) {
      acc[type].total += log.execTime;
      acc[type].count++;
    }
    return acc;
  }, {
    sharded: { total: 0, count: 0 },
    "non-sharded": { total: 0, count: 0 }
  });

  const latencyStats = logs.reduce((acc, log) => {
    const type = log.type?.toLowerCase();
    if (type === "sharded") {
      acc.sharded.finality += log.finalityTime || 0;
      acc.sharded.propagation += log.propagationLatency || 0;
      acc.sharded.count++;
    } else if (type === "non-sharded") {
      acc.nonSharded.finality += log.finalityTime || 0;
      acc.nonSharded.propagation += log.propagationLatency || 0;
      acc.nonSharded.count++;
    }
    return acc;
  }, {
    sharded: { finality: 0, propagation: 0, count: 0 },
    nonSharded: { finality: 0, propagation: 0, count: 0 },
  });
  
 const avgShardedFinality = latencyStats.sharded.count
    ? (latencyStats.sharded.finality / latencyStats.sharded.count).toFixed(2)
    : "N/A";
  const avgNonShardedFinality = latencyStats.nonSharded.count
    ? (latencyStats.nonSharded.finality / latencyStats.nonSharded.count).toFixed(2)
    : "N/A";

  const avgShardedPropagation = latencyStats.sharded.count
    ? (latencyStats.sharded.propagation / latencyStats.sharded.count).toFixed(2)
    : "N/A";
  const avgNonShardedPropagation = latencyStats.nonSharded.count
    ? (latencyStats.nonSharded.propagation / latencyStats.nonSharded.count).toFixed(2)
    : "N/A";

  const avgSharded = avgExec.sharded.count ? (avgExec.sharded.total / avgExec.sharded.count).toFixed(3) : "N/A";
  const avgNonSharded = avgExec["non-sharded"].count ? (avgExec["non-sharded"].total / avgExec["non-sharded"].count).toFixed(3) : "N/A";

  const shardedExecTime = logs
    .filter(log => log.type?.toLowerCase() === "sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.execTime ?? 0 }));

  const nonShardedExecTime = logs
    .filter(log => log.type?.toLowerCase() === "non-sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.execTime ?? 0 }));

  const shardedFinality = logs
    .filter(log => log.type?.toLowerCase() === "sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.finalityTime ?? 0 }));

  const nonShardedFinality = logs
    .filter(log => log.type?.toLowerCase() === "non-sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.finalityTime ?? 0 }));

  const shardedPropagation = logs
    .filter(log => log.type?.toLowerCase() === "sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.propagationLatency ?? 0 }));

  const nonShardedPropagation = logs
    .filter(log => log.type?.toLowerCase() === "non-sharded")
    .map((log, i) => ({ name: `Tx ${i + 1}`, value: log.propagationLatency ?? 0 }));

    

  return (
       <div className="bg-gray-900 p-6 rounded-lg mt-6 shadow-lg text-white">
      <div className="mt-4 text-sm bg-gray-800 p-4 rounded">
        <h4 className="text-lg font-semibold mb-2">Why Use Sharding?</h4>
        <ul className="list-disc list-inside space-y-1">
          <li><strong>Scalability:</strong> Transactions are processed in parallel across shards.</li>
          <li><strong>Performance:</strong> Lower average execution time when sharded logic is used.</li>
          <li><strong>Conflict Isolation:</strong> Sharding makes concurrency conflicts more localized and manageable.</li>
        </ul>
      </div>

      <h3 className="text-xl font-bold mb-4">Transaction Metrics</h3>

    {/* Average Execution Time */}
    <div className="mb-6 text-sm bg-gray-800 p-4 rounded">
      <h4 className="text-lg font-semibold mb-2">Average Execution Time</h4>
      <p className="text-sm"><span className="text-green-600">Sharded:</span> {avgSharded} ms</p>
      <p className="text-sm"><span className="text-red-600">Non-Sharded:</span> {avgNonSharded} ms</p>
    </div>

    {/* Finality, Propagation, TPS Boxes */}
    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 mb-6 text-sm">
      <div className="bg-gray-800 p-4 rounded">
        <h5 className="font-semibold text-sm text-yellow-400">Finality Time</h5>
        <p className="text-green-400 text-sm">Sharded: {avgShardedFinality} ms</p>
        <p className="text-red-400 text-sm">Non-Sharded: {avgNonShardedFinality} ms</p>
      </div>
      <div className="bg-gray-800 p-4 rounded">
        <h5 className="font-semibold text-sm text-pink-400">Propagation Latency</h5>
        <p className="text-green-400 text-sm">Sharded: {avgShardedPropagation} ms</p>
        <p className="text-red-400 text-sm">Non-Sharded: {avgNonShardedPropagation} ms</p>
      </div>
      <div className="bg-gray-800 p-4 rounded col-span-2">
        <h5 className="font-semibold text-sm text-purple-400">Avg TPS</h5>
        <p className="text-green-400 text-sm">Sharded: {avgShardedTPS} tx/s</p>
        <p className="text-red-400 text-sm">Non-Sharded: {avgNonShardedTPS} tx/s</p>
      </div>
    </div>


      <h3 className="text-xl font-bold mb-4">Transaction Metrics</h3>
          {/* ─────────────────────────────── Execution Time Toggle ─────────────────────────────── */}
    <div className="text-right mb-4">
      <button
        onClick={() => setShowCombinedExec(prev => !prev)}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded"
      >
        {showCombinedExec ? "View Split Execution Time" : "Compare Execution Time (Sharded vs Non-Sharded)"}
      </button>
    </div>

    {showCombinedExec ? (
      <div className="bg-gray-800 p-4 rounded mb-6">
        <h4 className="text-lg font-semibold mb-2 text-blue-300">Sharded vs Non-Sharded Execution Time</h4>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={combinedExecData}>
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip formatter={(value) => `${value?.toFixed(2)} ms`} />
            <Legend />
            <Line
              type="monotone"
              dataKey="shardedExec"
              stroke="#60a5fa"
              strokeWidth={2}
              dot
              connectNulls={true}
            />
            <Line
              type="monotone"
              dataKey="nonShardedExec"
              stroke="#3b82f6"
              strokeWidth={2}
              dot
              connectNulls={true}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    ) : (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-blue-300">Sharded Execution Time</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={shardedExecTime}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#60a5fa" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-blue-300">Non-Sharded Execution Time</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={nonShardedExecTime}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#3b82f6" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    )}

    {/* ─────────────────────────────── Finality Time Toggle ─────────────────────────────── */}
    <div className="text-right mb-4">
      <button
        onClick={() => setShowCombinedFinality(prev => !prev)}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded"
      >
        {showCombinedFinality ? "View Split Finality Time" : "Compare Finality Time (Sharded vs Non-Sharded)"}
      </button>
    </div>

    {showCombinedFinality ? (
      <div className="bg-gray-800 p-4 rounded mb-6">
        <h4 className="text-lg font-semibold mb-2 text-yellow-300">Sharded vs Non-Sharded Finality Time</h4>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={combinedFinalityData}>
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip formatter={(value) => `${value?.toFixed(2)} ms`} />
            <Legend />
            <Line
              type="monotone"
              dataKey="shardedFinality"
              stroke="#facc15"
              strokeWidth={2}
              dot
              connectNulls={true}
            />
            <Line
              type="monotone"
              dataKey="nonShardedFinality"
              stroke="#eab308"
              strokeWidth={2}
              dot
              connectNulls={true}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    ) : (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-yellow-300">Sharded Finality Time</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={shardedFinality}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#facc15" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-yellow-300">Non-Sharded Finality Time</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={nonShardedFinality}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#eab308" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    )}

    {/* ─────────────────────────────── Propagation Toggle ─────────────────────────────── */}
    <div className="text-right mb-4">
      <button
        onClick={() => setShowCombinedPropagation(prev => !prev)}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded"
      >
        {showCombinedPropagation ? "View Split Propagation Latency" : "Compare Propagation Latency (Sharded vs Non-Sharded)"}
      </button>
    </div>

    {showCombinedPropagation ? (
      <div className="bg-gray-800 p-4 rounded mb-6">
        <h4 className="text-lg font-semibold mb-2 text-pink-300">Sharded vs Non-Sharded Propagation Latency</h4>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={combinedPropagationData}>
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip formatter={(value) => `${value?.toFixed(2)} ms`} />
            <Legend />
            <Line
            type="monotone"
            dataKey="shardedPropagation"
            stroke="#fb7185"
            strokeWidth={2}
            dot
            connectNulls={true}
          />
          <Line
            type="monotone"
            dataKey="nonShardedPropagation"
            stroke="#f472b6"
            strokeWidth={2}
            dot
            connectNulls={true}
          />

          </LineChart>
        </ResponsiveContainer>
      </div>
    ) : (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-pink-300">Sharded Propagation Latency</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={shardedPropagation}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#fb7185" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <div className="bg-gray-800 p-4 rounded">
          <h4 className="text-lg font-semibold mb-2 text-pink-300">Non-Sharded Propagation Latency</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={nonShardedPropagation}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(val) => `${val.toFixed(2)} ms`} />
              <Line type="monotone" dataKey="value" stroke="#f472b6" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    )}
      {/* TPS Toggle */}
      <div className="text-right mb-4">
        <button
          onClick={() => setShowCombinedTPS(prev => !prev)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded"
        >
          {showCombinedTPS ? "View Split TPS" : "Compare TPS (Sharded vs Non-Sharded)"}
        </button>
      </div>

      {/* Combined TPS View */}
      {showCombinedTPS ? (
        <div className="bg-gray-800 p-4 rounded mb-6">
          <h4 className="text-lg font-semibold mb-2 text-cyan-300">Sharded vs Non-Sharded TPS</h4>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={combinedTPSData}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip formatter={(value) => `${value?.toFixed(2)} tx/s`} />
              <Legend />
              <Line type="monotone" dataKey="shardedTPS" stroke="#4ade80" strokeWidth={2} dot />
              <Line type="monotone" dataKey="nonShardedTPS" stroke="#f87171" strokeWidth={2} dot />
            </LineChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
          <div className="bg-gray-800 p-4 rounded">
            <h4 className="text-lg font-semibold mb-2 text-green-300">Sharded TPS</h4>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={shardedTPS}>
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip formatter={(value) => `${value.toFixed(2)} tx/s`} />
                <Line type="monotone" dataKey="tps" stroke="#4ade80" strokeWidth={2} dot />
              </LineChart>
            </ResponsiveContainer>
          </div>
          <div className="bg-gray-800 p-4 rounded">
            <h4 className="text-lg font-semibold mb-2 text-red-300">Non-Sharded TPS</h4>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={nonShardedTPS}>
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip formatter={(value) => `${value.toFixed(2)} tx/s`} />
                <Line type="monotone" dataKey="tps" stroke="#f87171" strokeWidth={2} dot />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Bar Chart Comparison */}
      <div className="mb-6">
        <h4 className="text-lg font-semibold mb-2">Execution Time by Type</h4>
        <ResponsiveContainer width="100%" height={250}>
          <BarChart data={[
            {
              type: "Sharded",
              execTime: avgExec.sharded.total,
              finalityTime: logs.filter(tx => tx.type?.toLowerCase() === "sharded")
                                .reduce((a, b) => a + (b.finalityTime || 0), 0),
            },
            {
              type: "Non-Sharded",
              execTime: avgExec["non-sharded"].total,
              finalityTime: logs.filter(tx => tx.type?.toLowerCase() === "non-sharded")
                                .reduce((a, b) => a + (b.finalityTime || 0), 0),
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
