import { createHash,createHmac,randomBytes,createPublicKey,createVerify,timingSafeEqual } from 'node:crypto';

const TTL_SECONDS=300;
export function oidcConfig(env=process.env){
 const issuer=env.OIDC_ISSUER?.replace(/\/$/,'');
 const id=env.OIDC_CLIENT_ID,secret=env.OIDC_CLIENT_SECRET;
 const redirect=env.OIDC_REDIRECT_URI;
 const adminGroup=env.OIDC_ADMIN_GROUP,viewerGroup=env.OIDC_VIEWER_GROUP;
 if(!issuer||!id||!redirect||!adminGroup||!viewerGroup)throw Error('OIDC issuer, client, callback and both groups required');
 if(!/^https:\/\//.test(issuer)||!/^https:\/\//.test(redirect))throw Error('OIDC requires HTTPS issuer and redirect');
 if(!/^[-a-zA-Z0-9_./:]+$/.test(issuer)||id.length>200||!adminGroup||!viewerGroup)throw Error('invalid OIDC configuration');
 if(!env.APP_SESSION_SECRET||env.APP_SESSION_SECRET.length<32)throw Error('APP_SESSION_SECRET missing');
 return {issuer,id,secret,redirect,adminGroup,viewerGroup,sessionKey:env.APP_SESSION_SECRET};
}
export function flowCookie(config,flow){
 const data=Buffer.from(JSON.stringify(flow)).toString('base64url');
 const mac=createHmac('sha256',config.sessionKey).update(data).digest('base64url');
 return data+'.'+mac;
}
export function verifyFlow(config,cookie,state,now=Date.now()){
 if(!cookie||cookie.length>2048||typeof state!=='string')throw Error('OIDC state missing');
 const [data,mac,...extra]=cookie.split('.');
 if(extra.length||!data||!mac)throw Error('invalid OIDC state');
 const expected=createHmac('sha256',config.sessionKey).update(data).digest();
 const signature=Buffer.from(mac,'base64url');
 if(signature.length!==expected.length||!timingSafeEqual(signature,expected))throw Error('OIDC state signature failed');
 const flow=JSON.parse(Buffer.from(data,'base64url').toString('utf8'));
 if(!flow||flow.state!==state||!(/^[A-Za-z0-9_-]{20,90}$/.test(state))||now-flow.created>TTL_SECONDS*1000||now<flow.created||!(/^[A-Za-z0-9_-]{20,90}$/.test(flow.nonce))||!(/^[A-Za-z0-9_-]{43,130}$/.test(flow.verifier)))throw Error('invalid or expired OIDC state');
 return flow;
}
export function beginFlow(config,now=Date.now()){
 const state=randomBytes(24).toString('base64url'),nonce=randomBytes(24).toString('base64url'),verifier=randomBytes(32).toString('base64url');
 return {flow:{state,nonce,verifier,created:now},cookie:flowCookie(config,{state,nonce,verifier,created:now})};
}
async function boundedJSON(url,options={}){
 const parsed=new URL(url);
 if(parsed.protocol!=='https:')throw Error('OIDC HTTPS endpoint required');
 const result=await fetch(url,{...options,redirect:'error',signal:AbortSignal.timeout(8000)});
 if(!result.ok)throw Error('OIDC endpoint rejected request');
 const text=await result.text();
 if(text.length>128000)throw Error('OIDC response too large');
 return JSON.parse(text);
}
export async function discovery(config){
 const doc=await boundedJSON(config.issuer+'/.well-known/openid-configuration');
 if(doc.issuer!==config.issuer)throw Error('OIDC issuer mismatch');
 for(const key of ['authorization_endpoint','token_endpoint','jwks_uri']){
  const url=new URL(doc[key]);
  if(url.protocol!=='https:'||url.origin!==new URL(config.issuer).origin)throw Error('OIDC endpoint origin mismatch');
 }
 return doc;
}
export function authorizeURL(config,doc,flow){
 const url=new URL(doc.authorization_endpoint);
 url.searchParams.set('client_id',config.id);
 url.searchParams.set('redirect_uri',config.redirect);
 url.searchParams.set('response_type','code');
 url.searchParams.set('scope','openid profile email');
 url.searchParams.set('state',flow.state);
 url.searchParams.set('nonce',flow.nonce);
 url.searchParams.set('code_challenge',createHash('sha256').update(flow.verifier).digest('base64url'));
 url.searchParams.set('code_challenge_method','S256');
 return url.toString();
}
export async function exchangeCode(config,doc,code,flow){
 if(typeof code!=='string'||code.length<5||code.length>2048)throw Error('Invalid OIDC code');
 const payload=new URLSearchParams({grant_type:'authorization_code',code,redirect_uri:config.redirect,client_id:config.id,code_verifier:flow.verifier});
 if(config.secret)payload.set('client_secret',config.secret);
 const token=await boundedJSON(doc.token_endpoint,{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:payload});
 if(typeof token.id_token!=='string'||token.id_token.length>32000)throw Error('ID token not returned');
 return token.id_token;
}
export function checkIDToken(jwt,config,nonce,jwks,now=Math.floor(Date.now()/1000)){
 if(typeof jwt!=='string'||jwt.length>32000)throw Error('ID token invalid');
 const parts=jwt.split('.');
 if(parts.length!==3)throw Error('Invalid JWT structure');
 let header,claims;
 try{header=JSON.parse(Buffer.from(parts[0],'base64url').toString());claims=JSON.parse(Buffer.from(parts[1],'base64url').toString())}catch{throw Error('Invalid JWT encoding')}
 if(header.alg!=='RS256'||typeof header.kid!=='string'||header.kid.length>160)throw Error('Unsupported JWT algorithm');
 const keys=jwks?.keys;
 if(!Array.isArray(keys)||keys.length>30)throw Error('Invalid JWKS');
 const key=keys.find(k=>k.kid===header.kid&&k.kty==='RSA'&&(!k.use||k.use==='sig'));
 if(!key)throw Error('Untrusted JWT key');
 const publicKey=createPublicKey({key,format:'jwk'});
 const verifier=createVerify('RSA-SHA256');verifier.update(parts[0]+'.'+parts[1]);verifier.end();
 if(!verifier.verify(publicKey,Buffer.from(parts[2],'base64url')))throw Error('OIDC signature invalid');
 if(claims.iss!==config.issuer||!(claims.aud===config.id||Array.isArray(claims.aud)&&claims.aud.includes(config.id))||claims.nonce!==nonce||!Number.isInteger(claims.exp)||claims.exp<=now||claims.exp>now+86400||!Number.isInteger(claims.iat)||claims.iat>now+60||claims.iat<now-86400||typeof claims.sub!=='string'||claims.sub.length<1||claims.sub.length>250)throw Error('OIDC claims rejected');
 const groups=Array.isArray(claims.groups)?claims.groups:Array.isArray(claims.roles)?claims.roles:[];
 const role=groups.includes(config.adminGroup)?'admin':groups.includes(config.viewerGroup)?'viewer':null;
 if(!role)throw Error('OIDC user has no approved group');
 const principal=createHmac('sha256',config.sessionKey).update('oidc:'+config.issuer+':'+claims.sub).digest('hex');
 return {role,principal};
}
export async function validateIdentity(config,doc,idToken,flow){
 const jwks=await boundedJSON(doc.jwks_uri);
 return checkIDToken(idToken,config,flow.nonce,jwks);
}
