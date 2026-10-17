import { NextResponse } from 'next/server';
import { COOKIE_NAME, sameOrigin } from '@/lib/auth';
export async function POST(req: Request) {
 if (!sameOrigin(req)) return NextResponse.json({error:'Invalid origin'}, {status:403});
 const res=NextResponse.json({ok:true});res.cookies.set(COOKIE_NAME,'',{maxAge:0,path:'/'});return res;
}
