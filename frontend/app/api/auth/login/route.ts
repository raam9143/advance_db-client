import { NextResponse } from 'next/server';
import { COOKIE_NAME, issueSession, sameOrigin, validPassword } from '@/lib/auth';
import { totpEnabled, verifyTotp } from '@/lib/totp.mjs';
import { consumeOTP, persistSession, loginAllowed, recordLoginFailure, clearLoginFailures } from '@/lib/session-store.mjs';
export const runtime = 'nodejs';
export async function POST(req: Request) {
 if (!sameOrigin(req)) return NextResponse.json({error:'Invalid origin'}, {status:403});
 if(process.env.OIDC_REQUIRED==='true')return NextResponse.json({error:'Local password login is disabled; use SSO'}, {status:403});
 if (Number(req.headers.get('content-length') || '0') > 4096) return NextResponse.json({error:'Payload too large'}, {status:413});
 // A single local-admin identity is throttled globally, not by client-supplied proxy headers.
 // With SESSION_REDIS_REQUIRED=true this fails closed and is shared across replicas.
 try{if(!await loginAllowed())return NextResponse.json({error:'Too many attempts; try again later'},{status:429})}
 catch{return NextResponse.json({error:'Authentication store unavailable'},{status:503})}
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
  try{await recordLoginFailure()}catch{return NextResponse.json({error:'Authentication store unavailable'},{status:503})}
  return NextResponse.json({error:'Invalid credentials'},{status:401});
 }
 if(totpEnabled()){
  try{
   if(!await consumeOTP(process.env.APP_TOTP_SECRET!,otp as string))return NextResponse.json({error:'Authenticator code already used'},{status:401});
  }catch{return NextResponse.json({error:'Session store unavailable'},{status:503})}
 }
 try{await clearLoginFailures()}catch{return NextResponse.json({error:'Authentication store unavailable'},{status:503})}
 const value=issueSession();
 try{await persistSession(value)}catch{return NextResponse.json({error:'Session store unavailable'},{status:503})}
 const res=NextResponse.json({ok:true});
 res.cookies.set(COOKIE_NAME,value,{httpOnly:true,secure:process.env.NODE_ENV==='production',sameSite:'strict',path:'/',maxAge:8*60*60});
 return res;
}
