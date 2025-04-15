// 'use client';

// import { usePathname } from 'next/navigation';
// import React from 'react';

// export default function BodyWrapper({ children }: { children: React.ReactNode }) {
//   const pathname = usePathname();

//   const isTransactionsPage = pathname === '/transactions';

//   return (
//     <body
//       style={{
//         height: '100vh',
//         overflow: isTransactionsPage ? 'auto' : 'hidden',
//       }}
//     >
//       {children}
//     </body>
//   );
// }
