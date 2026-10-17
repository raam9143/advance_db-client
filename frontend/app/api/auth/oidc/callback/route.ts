import { NextRequest,NextResponse } from 'next/server';
import { oidcConfig,verifyFlow,discovery,exchangeCode,validateIdentity } from '@/lib/oidc.mjs';
import { COOKIE_NAME,issueSession } from '@/lib/auth';
import { persistSession } from '@/lib/session-store.mjs';
export const runtime='nodejs';
export async function GET(req:NextRequest){
 try{
  const config=oidcConfig();
  const url=new URL(req.url);
  if(url.origin!==new URL(config.redirect).origin||url.pathname!==new URL(config.redirect).pathname)throw Error('unexpected redirect URL');
  if(url.searchParams.has('error'))throw Error('OIDC sign-in failed');
  const flow=verifyFlow(config,req.cookies.get('adb_oidc_flow')?.value,url.searchParams.get('state'));
  const doc=await discovery(config);
  const token=await exchangeCode(config,doc,url.searchParams.get('code')||'',flow);
  const identity=await validateIdentity(config,doc,token,flow);
  const value=issueSession(identity.role,identity.principal);
  await persistSession(value);
  const r=NextResponse.redirect(new URL('/',url.origin));
  r.cookies.set(COOKIE_NAME,value,{httpOnly:true,secure:true,sameSite:'strict',path:'/',maxAge:28800});
  r.cookies.set('adb_oidc_flow','',{path:'/api/auth/oidc',maxAge:0,secure:true});
  r.headers.set('Cache-Control','no-store');
  return r;
 }catch{
  const r=NextResponse.redirect(new URL('/login?error=oidc',req.url));
  r.cookies.set('adb_oidc_flow','',{path:'/api/auth/oidc',maxAge:0,secure:true});
  r.headers.set('Cache-Control','no-store');
  return r;
 }
}
