import type { Metadata } from 'next';
import './globals.css';
import './impeccable.css';
export const metadata: Metadata={title:'Advance DB Client',description:'Secure SQL workspace powered by Go and Next.js'};
export default function Layout({children}:{children:React.ReactNode}) {return <html lang="en"><body>{children}</body></html>}
