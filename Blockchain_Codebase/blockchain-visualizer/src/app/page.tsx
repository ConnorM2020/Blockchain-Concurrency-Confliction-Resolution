// src/app/page.tsx
"use client";

import BlockchainInteract from "./Blockchain_Interact";
import ExecutionPanel from "./ExecutionPanel";

export default function Home() {
  return (
    <div className="min-h-screen overflow-hidden p-8 pb-20 sm:p-20 flex-col items-center">
      <h1 className="text-3xl font-bold text-center mb-6">
        Spider-Web Blockchain View
      </h1>
      <ExecutionPanel />
      <BlockchainInteract />
    </div>
  );
}
