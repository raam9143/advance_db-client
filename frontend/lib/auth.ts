import { createHash, createHmac, randomBytes, timingSafeEqual } from 'node:crypto';
import { cookies } from 'next/headers';
import { sessionActive } from '@/lib/session-store.mjs';

export const COOKIE_NAME = 'adb_session';
const LIFE = 8 * 60 * 60;
export type Role = 'admin' | 'viewer';
export type Session = {role: Role; principal: string; expires: number; nonce: string};

function secret(): string {
 const key=process.env.APP_SESSION_SECRET;
 if(!key||key.length<32)throw Error('APP_SESSION_SECRET must be at least 32 characters');
 return key;
}
function signature(payload:string):string{return createHmac('sha256',secret()).update(payload).digest('base64url');}
export function issueSession(role:Role='admin',principal='local-admin',now=Math.floor(Date.now()/1000)):string{
 if(role!=='admin'&&role!=='viewer'||!(/^[a-zA-Z0-9_-]{1,90}$/.test(principal)))throw Error('invalid session principal');
 const value=`v2.${now+LIFE}.${randomBytes(16).toString('hex')}.${role}.${principal}`;
 return value+'.'+signature(value);
}
export function parseSession(value?:string):Session|null{
 if(!value||value.length>512)return null;
 const parts=value.split('.');
 if(parts.length!==6||parts[0]!=='v2'||!/^[0-9]{10}$/.test(parts[1])||!/^[a-f0-9]{32}$/.test(parts[2])||!['admin','viewer'].includes(parts[3])||!/^[a-zA-Z0-9_-]{1,90}$/.test(parts[4]))return null;
 const expires=Number(parts[1]),now=Math.floor(Date.now()/1000);
 if(expires<=now||expires>now+LIFE+60)return null;
 const got=Buffer.from(parts[5]),expected=Buffer.from(signature(parts.slice(0,5).join('.')));
 if(got.length!==expected.length||!timingSafeEqual(got,expected))return null;
 return {role:parts[3] as Role,principal:parts[4],expires,nonce:parts[2]};
}
export function verifySession(value?:string):boolean{return parseSession(value)!==null;}
export async function authorized():Promise<boolean>{
 const cookie=(await cookies()).get(COOKIE_NAME)?.value;
 if(!verifySession(cookie))return false;
 if(process.env.OIDC_REQUIRED==='true'&&parseSession(cookie)?.principal==='local-admin')return false;
 try{return await sessionActive(cookie)}catch{return false}
}
export function validPassword(submitted:string):boolean{
 const wanted=process.env.APP_PASSWORD;
 if(!wanted||wanted.length<12)throw Error('APP_PASSWORD must be at least 12 characters');
 const left=createHmac('sha256','adb-compare').update(submitted).digest();
 const right=createHmac('sha256','adb-compare').update(wanted).digest();
 return timingSafeEqual(left,right);
}
export function sameOrigin(request:Request):boolean{
 const origin=request.headers.get('origin'),host=request.headers.get('host');
 if(!origin||!host)return false;
 try{
  const actual=new URL(origin);
  if(process.env.ADB_PRODUCTION==='true'){
   const expected=new URL(process.env.APP_PUBLIC_ORIGIN||'');
   return expected.protocol==='https:'&&actual.origin===expected.origin&&host===expected.host;
  }
  return actual.host===host;
 }catch{return false}
}
export function sessionScope(value:string,apiToken:string):{id:string;proof:string;role:Role;roleProof:string}{
 const session=parseSession(value);
 if(!session)throw Error('Invalid session');
 const id=createHash('sha256').update(value).digest('hex');
 const proof=createHmac('sha256',apiToken).update('adb-session:'+id).digest('hex');
 const roleProof=createHmac('sha256',apiToken).update('adb-role:'+id+':'+session.role).digest('hex');
 return {id,proof,role:session.role,roleProof};
}
