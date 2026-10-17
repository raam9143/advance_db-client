import { createHmac, timingSafeEqual } from 'node:crypto';
const alphabet='ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
const used=new Map();
function parseBase32(secret){
 if(typeof secret!=='string')throw Error('APP_TOTP_SECRET missing');
 const normalized=secret.trim().replace(/\s/g,'').toUpperCase().replace(/=+$/,'');
 if(!/^[A-Z2-7]+$/.test(normalized))throw Error('APP_TOTP_SECRET must be Base32');
 let bits=0,acc=0;const out=[];
 for(const ch of normalized){
  acc=(acc<<5)|alphabet.indexOf(ch);bits+=5;
  if(bits>=8){bits-=8;out.push((acc>>bits)&255)}
 }
 if(out.length<20||out.length>128)throw Error('APP_TOTP_SECRET should decode to at least 20 bytes');
 return Buffer.from(out);
}
export function totpValue(secret,timeMs=Date.now(),digits=6){
 const key=parseBase32(secret);
 const step=Math.floor(timeMs/30000);
 const counter=Buffer.alloc(8);counter.writeBigUInt64BE(BigInt(step));
 const h=createHmac('sha1',key).update(counter).digest();
 const offset=h[h.length-1]&15;
 const code=((h[offset]&127)<<24)|(h[offset+1]<<16)|(h[offset+2]<<8)|h[offset+3];
 return String(code%(10**digits)).padStart(digits,'0');
}
export function verifyTotp(code,secret,timeMs=Date.now()){
 if(typeof code!=='string'||!/^\d{6}$/.test(code))return false;
 const key=parseBase32(secret);
 const identity=createHmac('sha256',key).update('admin totp').digest('hex');
 const step=Math.floor(timeMs/30000);
 let matched=-1;
 for(const drift of [-1,0,1]){
  const index=step+drift;if(index<0)continue;
  const expected=Buffer.from(totpValue(secret,index*30000));
  if(timingSafeEqual(expected,Buffer.from(code)))matched=Math.max(matched,index);
 }
 if(matched<0)return false;
 const last=used.get(identity);
 if(last!==undefined&&matched<=last)return false;
 used.set(identity,matched);
 return true;
}
export function totpEnabled(){
 return process.env.APP_REQUIRE_TOTP==='true';
}
