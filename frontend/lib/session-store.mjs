import { createHash,createHmac } from 'node:crypto';
import { createClient } from 'redis';

let pending;
function storeRequired(env=process.env){return env.SESSION_REDIS_REQUIRED==='true'||env.OIDC_REQUIRED==='true'}
function address(env=process.env){
 const value=env.SESSION_REDIS_URL;
 if(!value){
  if(storeRequired(env))throw Error('SESSION_REDIS_URL required for secure sessions');
  return null;
 }
 const url=new URL(value);
 if(url.protocol!=='rediss:'){
  if(url.protocol!=='redis:'||env.SESSION_REDIS_ALLOW_PLAINTEXT_LOOPBACK!=='true'||!['127.0.0.1','localhost','::1'].includes(url.hostname)){
   throw Error('session Redis must use verified TLS (except opted-in localhost)');
  }
 }
 return value;
}
async function client(){
 const url=address();
 if(!url)return null;
 if(!pending){
  pending=(async()=>{
   const connection=createClient({url,socket:{connectTimeout:4000,reconnectStrategy:()=>new Error('fail closed, no automatic reconnect')},disableOfflineQueue:true});
   connection.on('error',()=>{ /* authenticated requests fail on command error; never log Redis secrets */ });
   await connection.connect();
   return connection;
  })().catch(err=>{pending=undefined;throw err});
 }
 const result=await pending;
 if(!result.isReady){pending=undefined;throw Error('session Redis unavailable')}
 return result;
}
function fingerprint(value){return createHash('sha256').update(value).digest('hex')}
export function redisURLIsSafe(env=process.env){return address(env)}
export async function persistSession(cookie){
 const c=await client();if(!c)return true;
 const expiry=Number(cookie.split('.')[1]),ttl=expiry-Math.floor(Date.now()/1000);
 if(!Number.isInteger(ttl)||ttl<1||ttl>28800)throw Error('invalid session TTL');
 await c.set('adb:session:'+fingerprint(cookie),'1',{EX:ttl});
 return true;
}
export async function sessionActive(cookie){
 if(!cookie)return false;
 const c=await client();if(!c)return true;
 return await c.get('adb:session:'+fingerprint(cookie))==='1';
}
export async function revokeSession(cookie){
 if(!cookie)return true;
 const c=await client();if(!c)return true;
 await c.del('adb:session:'+fingerprint(cookie));
 return true;
}
export async function consumeOTP(secret,otp){
 const c=await client();if(!c)return true;
 const key=createHmac('sha256',secret).update('totp:'+otp).digest('hex');
 return await c.set('adb:totp:'+key,'1',{EX:120,NX:true})==='OK';
}
