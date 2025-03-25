'use client';

import React, { useState, useEffect, useMemo } from "react";
import ReactFlow, { MiniMap, Controls, Background, ReactFlowProvider, useEdgesState, useNodesState } from "reactflow";
import "reactflow/dist/style.css";
import dynamic from "next/dynamic";
import ExecutionPanel from "./ExecutionPanel";
import { useRouter } from "next/navigation"; 

const ReactFlowComponent = dynamic(() => import("reactflow"), { ssr: false });


const API_BASE = "http://localhost:8080";

export default function SpiderWebView() {
  const router = useRouter();
  const [nodes, setNodes] = useNodesState([]);
  const [edges, setEdges] = useEdgesState([]);
  const [selectedNodes, setSelectedNodes] = useState([]);
  const [selectedShard, setSelectedShard] = useState(1);
  const [shardOptions, setShardOptions] = useState([1, 2, 3, 4, 5]);
  const [shardModalOpen, setShardModalOpen] = useState(false);
  const [pendingTransactions, setPendingTransactions] = useState([]);
  const [transactionModalOpen, setTransactionModalOpen] = useState(false);
  const [transactionData, setTransactionData] = useState("");
  const [sourceNode, setSourceNode] = useState(null);
  const [targetNode, setTargetNode] = useState([]);
  

  const [transactionStatus, setTransactionStatus] = useState({});
  const [parallelModalOpen, setParallelModalOpen] = useState(false);
  const [parallelTransactions, setParallelTransactions] = useState([]);
  const [sidePanelOpen, setSidePanelOpen] = useState(false);
  const [logsDropdownOpen, setLogsDropdownOpen] = useState(false);
  const [logsOpen, setLogsOpen] = useState(false);
  const [transactionLogs, setTransactionLogs] = useState([]);
  const [selectedNode, setSelectedNode] = useState(null);
  const [crossShardModalOpen, setCrossShardModalOpen] = useState(false);

  useEffect(() => {
    fetchBlockchain();
    fetchTransactionLogs();
    if (pendingTransactions.length > 0) {
      const interval = setInterval(checkPendingTransactions, 2000);
      return () => clearInterval(interval);
    }
  }, [pendingTransactions]);  


  const fetchBlockchain = async () => {
    try {
      const response = await fetch(`${API_BASE}/blockchain`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      const data = await response.json();
      if (!Array.isArray(data) || data.length === 0) {
        console.warn("⚠️ Blockchain data is empty or malformed. Skipping update.");
        return; // Don't call formatSpiderWebData if data is invalid
      }
  
      formatSpiderWebData(data);
  
      for (const transactionID of pendingTransactions) {
        const status = await checkTransactionStatus(transactionID);
        if (status === "completed") {
          setTransactionStatus((prev) => ({ ...prev, [transactionID]: "Completed" }));
  
          setNodes((prevNodes) =>
            prevNodes.map((node) =>
              node.id === sourceNode?.id
                ? { ...node, style: { background: "blue" } }
                : targetNode.some((t) => t.id === node.id)
                ? { ...node, style: { background: "lightgreen" } }
                : node
            )
          );
  
          // Reset node selections only after UI has updated
          setTimeout(() => {
            setSourceNode(null);
            setTargetNode([]);
          }, 1000);
        }
      }
    } catch (err) {
      console.error("Error fetching blockchain data:", err);
    }
  };  

  const sendTransaction = async (type = "non-sharded") => {
    if (!sourceNode || !targetNode.length || !transactionData.trim()) {
      alert("Please select a source, at least one target, and enter data.");
      return;
    }
    const sourceShard = sourceNode?.data?.shard_id;
  
    // Prevent transaction to self
    const selfTargets = targetNode.filter(
      (target) => target.id === sourceNode.id
    );
    if (selfTargets.length > 0) {
      alert("❌ Cannot send a transaction from a node to itself.");
      return;
    }
    // Ensure all targets are in the same shard as the source
    const invalidTargets = targetNode.filter(
      (target) => target?.data?.shard_id !== sourceShard
    );
  
    if (invalidTargets.length > 0) {
      const targetInfo = invalidTargets
        .map((t) => `${t.data?.label} (Shard ${t.data?.shard_id})`)
        .join(", ");
      alert(`❌ Invalid Non-Sharded Transaction:\nAll nodes must be in the SAME shard.\n` +
        `Source is in Shard ${sourceShard}, but mismatched targets:\n${targetInfo}` );
      return;
    }
  
    try {
      const responses = await Promise.all(
        targetNode.map(async (target) => {
          const transaction = {
            source: Number(sourceNode.id),
            target: Number(target.id),
            data: transactionData,
            is_sharded: type === "sharded",
          };
  
          console.log("Sending transaction:", transaction);
  
          const res = await fetch(`${API_BASE}/addTransaction`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(transaction),
          });
  
          const text = await res.text();
          if (!res.ok) {
            console.error("❌ Backend responded with error:", text);
            throw new Error(`HTTP Error: ${res.status}`);
          }
          const parsed = JSON.parse(text);
          return parsed.transactionID || null;
        })
      );
  
      const validTransactionIDs = responses.filter((id) => id);
      if (validTransactionIDs.length > 0) {
        setPendingTransactions((prev) => [...prev, ...validTransactionIDs]);
        alert(`✅ ${validTransactionIDs.length} Non-Sharded transaction(s) sent!`);
      } else {
        alert("⚠️ No valid transactions were confirmed by the backend.");
      }
  
      setTransactionModalOpen(false);
      setSourceNode(null);
      setTargetNode([]);
      setTransactionData("");
  
    } catch (error) {
      console.error("❌ Error sending transaction(s):", error);
      alert("Transaction(s) failed to send.");
    }
  };

  const fetchTransactionLogs = async () => {
    try {
      const response = await fetch(`${API_BASE}/transactionLogs`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
      const data = await response.json();
      setTransactionLogs(data.logs.slice(-10)); // Show last 10 transactions
    } catch (err) {
      console.error("Error fetching transaction logs:", err);
    }
  };
  
  const checkPendingTransactions = async () => {
    if (pendingTransactions.length === 0) return;
  
    const updatedPending = [];
    for (const transactionID of pendingTransactions) {
      try {
        const response = await fetch(`${API_BASE}/transactionStatus/${transactionID}`);
        if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
        const data = await response.json();
        if (data.status === "completed") {
          setTransactionStatus((prev) => ({ ...prev, [transactionID]: "Completed" }));
          fetchBlockchain(); // Refresh UI when transaction is done
        } else {
          updatedPending.push(transactionID); // Keep tracking if not completed
        }
      } catch (err) {
        console.error(`Error checking status for ${transactionID}:`, err);
      }
    }
    setPendingTransactions(updatedPending);
  };
  
  const checkTransactionStatus = async (transactionID) => {
    try {
      const response = await fetch(`${API_BASE}/transactionStatus/${transactionID}`);
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      const data = await response.json();
      return data.status;
    } catch (error) {
      console.error("Error checking transaction status:", error);
      return "Unknown";
    }
  };
  const sendShardedTransactions = async () => {
    if (parallelTransactions.length === 0) {
      alert("Please enter at least one transaction.");
      return;
    }
  
    try {
      const formattedTransactions = parallelTransactions.map(tx => ({
        source: tx.source.split(",").map(node => node.trim()), // Convert CSV to array
        target: tx.target.split(",").map(node => node.trim()),
        data: tx.data.trim(),
      }));
  
      const shardSize = 3;
      const shards = [];
      for (let i = 0; i < formattedTransactions.length; i += shardSize) {
        shards.push(formattedTransactions.slice(i, i + shardSize));
      }
  
      // Send each shard in parallel
      const promises = shards.map(shard =>
        fetch(`${API_BASE}/shardTransactions`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ transactions: shard }),
        }).then(response => response.json())
          .then(data => updatePeerNodes(data.transactionIDs)) // Notify peers
      );
  
      await Promise.all(promises);
      alert("Transactions successfully sent and peers updated!");
  
      setParallelTransactions([]);
      setParallelModalOpen(false);
    } catch (error) {
      console.error("Error sending sharded transactions:", error);
    }
  };
  
  // Function to notify all peer nodes about the new transactions
  const updatePeerNodes = async (transactionIDs) => {
    try {
      await fetch(`${API_BASE}/updatePeers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ transactionIDs }),
      });
      console.log("✅ Peers successfully updated with transaction history.");
    } catch (error) {
      console.error("Error updating peers:", error);
    }
  };
    const formatSpiderWebData = (blocks) => {
      let newNodes = [];
      let newEdges = [];
      let shardMap = {};

      blocks.forEach((block) => {
          if (!shardMap[block.shard_id]) {
              shardMap[block.shard_id] = [];
          }
          shardMap[block.shard_id].push(block);
      });
      let shardColors = [
          "rgba(255, 87, 51, 0.3)",  // Light red
          "rgba(51, 255, 87, 0.3)",  // Light green
          "rgba(51, 133, 255, 0.3)", // Light blue
          "rgba(239, 239, 8, 0.3)", // Light yellow
          "rgba(61, 10, 228, 0.3)"  // Light purpose
      ];

      let shardBubbles = [];

      Object.entries(shardMap).forEach(([shardId, shardBlocks], index) => {
          let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
          let angleStep = (2 * Math.PI) / shardBlocks.length;
          let radius = 150 + shardBlocks.length * 10;

          shardBlocks.forEach((block, i) => {
              let x = 400 + Math.cos(angleStep * i) * radius + index * 400;
              let y = 300 + Math.sin(angleStep * i) * radius;

              let shardIndex = parseInt(block.shard_id) || 0;
              let assignedColor = shardColors[shardIndex % shardColors.length];

              newNodes.push({
                id: block.index.toString(),
                data: {
                  label: `Block ${block.index}`,
                  ...block,
                  color: assignedColor.replace("0.3", "1"), //  Store the original color here
                },
                position: { x, y },
                style: {
                  background: assignedColor.replace("0.3", "1"),
                  color: "#fff",
                  borderRadius: "5px",
                  padding: "10px",
                  cursor: "pointer",
                  border: "none",
                },
                draggable: true,
              });
              
              
              // Track bounding box
              minX = Math.min(minX, x);
              minY = Math.min(minY, y);
              maxX = Math.max(maxX, x);
              maxY = Math.max(maxY, y);

              if (block.previous_hash !== "0") {
                  newEdges.push({
                      id: `e${block.index}`,
                      source: (block.index - 1).toString(),
                      target: block.index.toString(),
                      animated: true,
                      style: { stroke: "#999" },
                  });
              }
          });

          // Create a bounding box for each shard
          let width = maxX - minX + 50;
          let height = maxY - minY + 50;

          shardBubbles.push({
              id: `shard-${shardId}`,
              position: { x: minX - 25, y: minY - 25 },
              data: { label: `Shard ${shardId}` },
              style: {
                  width: width,
                  height: height,
                  background: shardColors[shardId % shardColors.length],
                  borderRadius: "50%",
                  position: "absolute",
                  zIndex: -1,
              },
              type: "shardBubble",
              draggable: false,
          });
      });

      setNodes([...shardBubbles, ...newNodes]);
      setEdges(newEdges);
  };
  const handleNodeClick = (event, node) => {
    // Prevent Shard clicking 
    if (node.type === "shardBubble") {
      console.warn("Cannot select the shard ID");
      return;
    }
  
    // First click = set source node
    if (!sourceNode) {
      setSourceNode(node);
  
      // Highlight source node
      setNodes((prevNodes) =>
        prevNodes.map((n) =>
          n.id === node.id ? { ...n, style: { ...n.style, background: "blue" } } : n
        )
      );
    } else {
      // Holding Ctrl: toggle multi-target selection
      const isCtrlPressed = event.ctrlKey || event.metaKey;
  
      if (isCtrlPressed) {
        setTargetNode((prevTargets) => {
          const exists = prevTargets.find((t) => t.id === node.id);
  
          if (node.id === sourceNode.id) {
            console.warn("Cannot select source node as target!");
            return prevTargets;
          }
  
          let updatedTargets;
          if (exists) {
            // Remove if already selected
            updatedTargets = prevTargets.filter((t) => t.id !== node.id);
          } else {
            // Add if new
            updatedTargets = [...prevTargets, node];
          }
  
          // Update visual feedback
          setNodes((prevNodes) =>
            prevNodes.map((n) => {
              if (n.id === node.id) {
                return {
                  ...n,
                  style: {
                    ...n.style,
                    background: exists ? node.data?.color || "#444" : "green", // Toggle green if newly added
                  },
                };
              }
              return n;
            })
          );
  
          return updatedTargets;
        });
      } else {
        // Not holding Ctrl: reset source and re-select
        setSourceNode(node);
        setTargetNode([]);
  
        // Reset styles, highlight only the new source node
        setNodes((prevNodes) =>
          prevNodes.map((n) => {
            if (n.type === "shardBubble") return n;
            if (n.id === node.id) {
              return { ...n, style: { ...n.style, background: "blue", color: "#fff" } };
            }
            return {
              ...n,
              style: { ...n.style, background: n.data?.color || "#444", color: "#fff" },
            };
          })
        );
      }
    }
  };

  const toggleNodeSelection = (nodeId) => {
    const numericNodeId = Number(nodeId); // Ensure nodeId is a number
    setSelectedNodes((prev) =>
      prev.includes(numericNodeId)
        ? prev.filter((id) => id !== numericNodeId) // Remove if already selected
        : [...prev, numericNodeId] // Add if not selected
    );
  };
  
  const updateParallelTransaction = (index, field, value) => {
    setParallelTransactions((prev) => {
        const updatedTransactions = [...prev];

        // Validation range checking
        if (field === "target" || field === "source") {
            let nodeValue = Number(value);
            const maxNodes = nodes.length; // Get total nodes count

            // Ensure value is at least 1 and cycles if exceeding maxNodes
            if (nodeValue < 1) {
                nodeValue = 1;
            } else if (nodeValue > maxNodes-1) {
                nodeValue = 1; // Restart from 1 when exceeding available nodes
            }

            updatedTransactions[index] = { ...updatedTransactions[index], [field]: nodeValue };
        } else {
            updatedTransactions[index] = { ...updatedTransactions[index], [field]: value };
        }

        return updatedTransactions;
    });
};

const sendParallelTransactions = async () => {
  const transactionsToSend = parallelTransactions.length > 0
    ? parallelTransactions
    : [{
        source: sourceNode?.id,
        target: targetNode.map(n => n.id),
        data: transactionData.trim(),
      }];

  const validTransactions = transactionsToSend.filter(
    tx => tx.source && tx.target.length > 0 && tx.data
  );
  // Prevent any self-transactions
  const hasSelfTransaction = validTransactions.some(tx =>
    tx.target.includes(Number(tx.source))
  );
  if (hasSelfTransaction) {
    alert("❌ One or more transactions attempt to send to the same node (self-transaction). This is not allowed.");
    return;
  }
  if (validTransactions.length === 0) {
    alert("Please enter at least one transaction.");
    return;
  }

  try {
    const formattedTransactions = validTransactions.map(tx => ({
      source: Array.isArray(tx.source) ? tx.source : [Number(tx.source)],
      target: tx.target.map(id => Number(id)),
      data: tx.data.trim(),
    }));

    const shardSize = 3;
    const shards = [];
    for (let i = 0; i < formattedTransactions.length; i += shardSize) {
      shards.push(formattedTransactions.slice(i, i + shardSize));
    }

    const promises = shards.map((shard, index) => {
      console.log(`🚀 Sending Sharded Transaction Group #${index + 1}:`);
      shard.forEach(tx => {
        console.log(`→ From Node ${tx.source} to Node(s) ${tx.target.join(", ")} | Data: "${tx.data}"`);
      });

      return fetch("http://localhost:8080/shardTransactions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ transactions: shard }),
      }).then(res => res.json());
    });

    const responses = await Promise.all(promises);
    const transactionIDs = responses.flatMap(res => res.transactionIDs || []);

    setPendingTransactions(prev => [...prev, ...transactionIDs]);
    setParallelTransactions([]);
    setTransactionData("");
    setSourceNode(null);
    setTargetNode([]);
    alert("✅ Sharded transaction(s) sent successfully!");
    
  } catch (error) {
    console.error("Error sending parallel transactions:", error);
    alert("❌ Failed to send transaction(s).");
  }
};

  // Define the custom node type
  const shardBubbleNode = ({ data }) => {
    return (
      <div
        style={{
          background: data.color || "gray",
          padding: "10px",
          borderRadius: "50%",
          textAlign: "center",
          color: "#fff",
        }}
      >
        {data.label}
      </div>
    );
  };

  // Inside your component:
  const nodeTypes = useMemo(() => ({
    shardBubble: shardBubbleNode,
  }), []);

  const confirmShardCreation = async () => {
    if (selectedNodes.length === 0) {
      alert("Please select nodes to assign to a shard.");
      return;
    }
  
    try {
      const response = await fetch(`${API_BASE}/assignNodesToShard`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ shard_id: selectedShard, nodes: selectedNodes.map(Number) }),
      });
  
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      console.log(`Nodes ${selectedNodes} assigned to Shard ${selectedShard}`);
      fetchBlockchain();  // Refresh the visualization
      setSelectedNodes([]); 
      setShardModalOpen(false);
    } catch (err) {
      console.error("Error assigning nodes to shard:", err);
    }
  };
  


  const resetBlockchain = async () => {
    try {
      const response = await fetch(`${API_BASE}/resetBlockchain`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
  
      if (!response.ok) throw new Error(`HTTP Error: ${response.status}`);
  
      console.log("✅ Blockchain reset successfully.");
      fetchBlockchain(); // Refresh UI after reset
    } catch (err) {
      console.error("Error resetting blockchain:", err);
    }
  };
  
  return (
    <ReactFlowProvider>
      <div className="w-screen h-screen flex bg-black text-white">
      {/* Sidebar Toggle Button */}
      <button
        className="absolute top-4 left-10 bg-blue-600 text-white px-4 py-2 rounded z-10"
        onClick={() => setSidePanelOpen(!sidePanelOpen)}
        >
        {sidePanelOpen ? "← Close Panel" : "→ Open Transactions"}
        </button>

        {/* Side Panel */}
        <div
          className={`absolute top-0 left-0 h-full bg-gray-800 p-6 shadow-lg transition-transform duration-300 ${
            sidePanelOpen ? "translate-x-0" : "-translate-x-full"
          }`}
          style={{ width: "250px" }}
        >
          <h2 className="text-lg font-bold mb-4 mt-12">Transaction Options</h2>

          <button className="w-full px-4 py-2 mb-2 bg-purple-600 text-white rounded" onClick={() => setShardModalOpen(true)}>
            Create New Shard
          </button>
          <button className="w-full px-4 py-2 bg-green-600 text-white rounded" onClick={() => setParallelModalOpen(true)}>
            Parallel Transactions
          </button>

          <button className="w-full px-4 py-2 bg-blue-600 text-white rounded" onClick={fetchBlockchain}>
            Refresh Blockchain
          </button>
          <button className="w-full px-4 py-2 bg-red-600 text-white rounded" onClick={resetBlockchain}>
            Reset Blockchain
          </button>
          <button
            onClick={() => router.push("/transactions")}
            className="w-full px-6 py-3 bg-indigo-600 hover:bg-blue-700 text-white font-bold rounded-lg mt-6 shadow-md transition duration-200 ease-in-out"
          > View All Transactions
          </button>
        </div>

        {/* Main Content */}
        <div className="w-full h-full relative mt-4">
        <ReactFlowComponent nodes={nodes} edges={edges} nodeTypes={nodeTypes} onNodeClick={handleNodeClick} >
            <MiniMap />
            <Controls />
            <Background />
          </ReactFlowComponent>
        </div>

        {selectedNode && selectedNode.data && (
        <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
          <h2 className="text-xl font-bold mb-4 text-white">Node Details</h2>

          <p className="text-white"><strong>Name:</strong> {selectedNode?.data?.label || "N/A"}</p>
          <p className="text-white"><strong>Shard:</strong> {selectedNode?.data?.shard || "Unassigned"}</p>
          <p className="text-white"><strong>Peers:</strong> {selectedNode?.data?.peers?.join(", ") || "None"}</p>
          <p className="text-white"><strong>Hash:</strong> {selectedNode?.data?.hash || "N/A"}</p>
          <p className="text-white"><strong>Transactions:</strong> {selectedNode?.data?.transactions?.length || 0}</p>
          <p className="text-white"><strong>Amount:</strong> {selectedNode?.data?.amount || "N/A"}</p>

          <button onClick={() => setSelectedNode(null)} className="px-4 py-2 bg-gray-500 text-white rounded mt-4">
            Close
          </button>
        </div>
      )}
        {/* Transaction Status Overlay */}
        {Object.keys(transactionStatus || {}).length > 0 &&
          Object.keys(transactionStatus).map((txID) => (
            <div key={txID} className="absolute top-4 right-4 bg-gray-800 text-white px-4 py-2 rounded">
              {txID}: {transactionStatus[txID]}
            </div>
          ))}
  
        {/* Transaction Modal */}
        {transactionModalOpen && (
          <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
            <h2 className="text-xl font-bold mb-4">Send Transaction</h2>
            <textarea
              className="w-full h-20 p-2 bg-gray-700 text-white rounded"
              value={transactionData}
              onChange={(e) => setTransactionData(e.target.value)}
            />
            <button onClick={sendTransaction} className="px-4 py-2 bg-green-600 text-white rounded mt-4">
              Send
            </button>
          </div>
        )}
  
       {/* Shard Selection Modal */}
      {shardModalOpen && (
        <div className="absolute top-1/3 left-1/2 transform -translate-x-1/2 bg-gray-800 p-6 rounded-lg shadow-lg w-96">
          <h2 className="text-xl font-bold mb-4">Assign Nodes to a Shard</h2>

          {/* Dropdown for selecting Shard ID */}
          <label className="block text-white mb-2">Select Shard ID:</label>
          <select
            value={selectedShard}
            onChange={(e) => setSelectedShard(Number(e.target.value))}
            className="w-full p-2 mb-4 bg-gray-700 text-white rounded"
          >
            {shardOptions.map((shard) => (
              <option key={shard} value={shard}>
                Shard {shard}
              </option>
            ))}
          </select>

          {/* Checkboxes for selecting blocks (excluding shards) */}
          <div className="max-h-40 overflow-y-auto border p-2 rounded bg-gray-700">
            {nodes
              .filter(node => !node.data.label.includes("Shard")) // Exclude shard names
              .map((node) => (
                <label key={node.id} className="flex items-center space-x-2 mb-2">
                  <input
                    type="checkbox"
                    checked={selectedNodes.includes(Number(node.id))}
                    onChange={() => toggleNodeSelection(node.id)}
                  />
                  <span>{node.data.label}</span>
                </label>
              ))}
          </div>

          <div className="flex justify-between mt-4">
            {/* Back Button */}
            <button onClick={() => setShardModalOpen(false)} className="px-4 py-2 bg-gray-500 text-white rounded">
              Back
            </button>

            <button onClick={confirmShardCreation} className="px-4 py-2 bg-green-600 text-white rounded">
              Assign to Shard
            </button>
          </div>
        </div>
      )}
      
      </div>
      {sourceNode && (
      <ExecutionPanel
        sourceNode={sourceNode}
        targetNode={targetNode}
        setSourceNode={setSourceNode}
        setTargetNode={setTargetNode}
        setNodes={setNodes} 
        transactionData={transactionData}
        setTransactionData={setTransactionData}
        sendTransaction={sendTransaction}
        sendParallelTransactions={sendParallelTransactions}
      />
    )}
    </ReactFlowProvider>
  );
}  