import type { Metadata } from 'next';
import './globals.css';
import './impeccable.css';
export const metadata: Metadata={title:'Croft',description:'A careful hold on every database.',icons:{icon:'/brand/croft-icon.svg'}};
export default function Layout({children}:{children:React.ReactNode}) {return <html lang="en"><body>{children}</body></html>}
