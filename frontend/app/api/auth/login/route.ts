import { NextResponse } from 'next/server';
import { COOKIE_NAME, issueSession, sameOrigin, validPassword } from '@/lib/auth';
import { totpEnabled, verifyTotp } from '@/lib/totp.mjs';
import { consumeOTP, persistSession } from '@/lib/session-store.mjs';
export const runtime = 'nodejs';
const attempts = new Map<string, { count: number; until: number }>();
export async function POST(req: Request) {
 if (!sameOrigin(req)) return NextResponse.json({error:'Invalid origin'}, {status:403});
 if(process.env.OIDC_REQUIRED==='true')return NextResponse.json({error:'Local password login is disabled; use SSO'}, {status:403});
 if (Number(req.headers.get('content-length') || '0') > 4096) return NextResponse.json({error:'Payload too large'}, {status:413});
 const ip = req.headers.get('x-forwarded-for')?.split(',')[0]?.trim() || 'unknown';
 const now=Date.now(), entry=attempts.get(ip);
 if (entry && entry.count>=5 && entry.until>now) return NextResponse.json({error:'Too many attempts; try again later'}, {status:429});
 const raw = await req.text();
 if(raw.length>4096)return NextResponse.json({error:'Payload too large'},{status:413});
 const body: unknown = (()=>{try{return JSON.parse(raw)}catch{return null}})();
 const password = body && typeof body==='object' && 'password' in body ? (body as {password:unknown}).password : null;
 const otp = body && typeof body==='object' && 'otp' in body ? (body as {otp:unknown}).otp : null;
 if(totpEnabled()&&(!process.env.APP_TOTP_SECRET||process.env.APP_TOTP_SECRET.length<32)){
  return NextResponse.json({error:'Authenticator configuration missing'},{status:503});
 }
 if (typeof password !== 'string' || password.length>2048 || !validPassword(password) ||
   (totpEnabled()&&(typeof otp!=='string'||!verifyTotp(otp,process.env.APP_TOTP_SECRET!)))) {
  const n=entry && entry.until>now ? entry.count+1 : 1;
  attempts.set(ip,{count:n,until:now+15*60*1000});
  return NextResponse.json({error:'Invalid credentials'},{status:401});
 }
 if(totpEnabled()){
  try{
   if(!await consumeOTP(process.env.APP_TOTP_SECRET!,otp as string))return NextResponse.json({error:'Authenticator code already used'},{status:401});
  }catch{return NextResponse.json({error:'Session store unavailable'},{status:503})}
 }
 attempts.delete(ip);
 const value=issueSession();
 try{await persistSession(value)}catch{return NextResponse.json({error:'Session store unavailable'},{status:503})}
 const res=NextResponse.json({ok:true});
 res.cookies.set(COOKIE_NAME,value,{httpOnly:true,secure:process.env.NODE_ENV==='production',sameSite:'strict',path:'/',maxAge:8*60*60});
 return res;
}
