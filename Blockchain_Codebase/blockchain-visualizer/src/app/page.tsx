"use client"; // Ensures this runs as a client-side component

import BlockchainInteract from "./Blockchain_Interact";

export default function Home() {
  return (
    <div className="min-h-screen p-8 pb-20 sm:p-20 flex flex-col items-center">
      <h1 className="text-3xl font-bold text-center mb-6"> </h1>
      
      {/* Blockchain Visualization Component */}
      <BlockchainInteract />
    </div>
  );
}
