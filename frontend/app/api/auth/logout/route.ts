import { NextResponse } from 'next/server';
import { COOKIE_NAME, sameOrigin } from '@/lib/auth';
import { revokeSession } from '@/lib/session-store.mjs';
export async function POST(req: Request) {
 if (!sameOrigin(req)) return NextResponse.json({error:'Invalid origin'}, {status:403});
 let ok=true;
 try{await revokeSession(req.headers.get('cookie')?.match(/(?:^|;\\s*)adb_session=([^;]*)/)?.[1])}catch{ok=false}
 const res=NextResponse.json(ok?{ok:true}:{error:'Unable to revoke session server-side'},{status:ok?200:503});
 res.cookies.set(COOKIE_NAME,'',{maxAge:0,path:'/'});
 return res;
}
