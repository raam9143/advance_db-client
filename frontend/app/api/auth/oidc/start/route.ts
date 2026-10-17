import { NextResponse } from 'next/server';
import { oidcConfig,discovery,beginFlow,authorizeURL } from '@/lib/oidc.mjs';
export const runtime='nodejs';
export async function GET(){
 try{
  const config=oidcConfig(),doc=await discovery(config);
  const {flow,cookie}=beginFlow(config);
  const r=NextResponse.redirect(authorizeURL(config,doc,flow));
  r.cookies.set('adb_oidc_flow',cookie,{httpOnly:true,secure:true,sameSite:'lax',path:'/api/auth/oidc',maxAge:300});
  r.headers.set('Cache-Control','no-store');
  return r;
 }catch{return NextResponse.json({error:'OIDC provider configuration or availability failed'},{status:503})}
}
