import * as React from "react";

export function Card({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border bg-white shadow-sm p-4">{children}</div>
  );
}

export function CardContent({ children }: { children: React.ReactNode }) {
  return <div className="p-4">{children}</div>;
}
