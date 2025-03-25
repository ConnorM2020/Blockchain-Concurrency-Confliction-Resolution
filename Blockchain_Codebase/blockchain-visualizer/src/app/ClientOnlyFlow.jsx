// components/ClientOnlyFlow.jsx
'use client';

import dynamic from 'next/dynamic';
import { MiniMap, Controls, Background } from 'reactflow';

// Dynamically import only the actual ReactFlow component
const ReactFlow = dynamic(() =>
  import('reactflow').then(mod => mod.ReactFlow), { ssr: false }
);

export default function ClientOnlyFlow({ nodes, edges, nodeTypes, onNodeClick }) {
  return (
    <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} onNodeClick={onNodeClick}>
      <MiniMap />
      <Controls />
      <Background />
    </ReactFlow>
  );
}
