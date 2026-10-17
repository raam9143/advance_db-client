import { NextResponse } from 'next/server';
import { COOKIE_NAME, issueSession, sameOrigin, validPassword } from '@/lib/auth';
export const runtime = 'nodejs';
const attempts = new Map<string, { count: number; until: number }>();
export async function POST(req: Request) {
 if (!sameOrigin(req)) return NextResponse.json({error:'Invalid origin'}, {status:403});
 if (Number(req.headers.get('content-length') || '0') > 4096) return NextResponse.json({error:'Payload too large'}, {status:413});
 const ip = req.headers.get('x-forwarded-for')?.split(',')[0]?.trim() || 'unknown';
 const now=Date.now(), entry=attempts.get(ip);
 if (entry && entry.count>=5 && entry.until>now) return NextResponse.json({error:'Too many attempts; try again later'}, {status:429});
 const raw = await req.text();
 if(raw.length>4096)return NextResponse.json({error:'Payload too large'},{status:413});
 const body: unknown = (()=>{try{return JSON.parse(raw)}catch{return null}})();
 const password = body && typeof body==='object' && 'password' in body ? (body as {password:unknown}).password : null;
 if (typeof password !== 'string' || password.length>2048 || !validPassword(password)) {
  const n=entry && entry.until>now ? entry.count+1 : 1;
  attempts.set(ip,{count:n,until:now+15*60*1000});
  return NextResponse.json({error:'Invalid credentials'},{status:401});
 }
 attempts.delete(ip);
 const res=NextResponse.json({ok:true});
 res.cookies.set(COOKIE_NAME,issueSession(),{httpOnly:true,secure:process.env.NODE_ENV==='production',sameSite:'strict',path:'/',maxAge:8*60*60});
 return res;
}
