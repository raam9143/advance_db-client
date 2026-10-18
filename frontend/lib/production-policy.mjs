// Deploy-time fail-closed checks, deliberately skipped for development and tests.
import {oidcConfig} from './oidc.mjs';
import {redisURLIsSafe} from './session-store.mjs';
function check(env){
 const failures=[];
 const missing=(k)=>{if(!env[k])failures.push(k+' is required')};
 const needed=['APP_SESSION_SECRET','API_TOKEN','APP_PUBLIC_ORIGIN','SESSION_REDIS_URL','OIDC_ISSUER','OIDC_CLIENT_ID','OIDC_REDIRECT_URI','OIDC_ADMIN_GROUP','OIDC_VIEWER_GROUP','OIDC_REQUIRED_AMR'];
 for(const k of needed)missing(k);
 if(env.OIDC_REQUIRED!=='true')failures.push('OIDC_REQUIRED must be true');
 if(env.SESSION_REDIS_REQUIRED!=='true')failures.push('SESSION_REDIS_REQUIRED must be true');
 if(env.SESSION_REDIS_ALLOW_PLAINTEXT_LOOPBACK==='true')failures.push('plaintext Redis exceptions prohibited');
 if(env.APP_REQUIRE_TOTP==='true'&&!env.APP_TOTP_SECRET)failures.push('APP_TOTP_SECRET required when enabled');
 if(env.APP_PASSWORD)failures.push('APP_PASSWORD must not be configured with production SSO');
 if(env.APP_SESSION_SECRET&&env.APP_SESSION_SECRET.length<32)failures.push('APP_SESSION_SECRET too short');
 if(env.API_TOKEN&&env.API_TOKEN.length<32)failures.push('API_TOKEN too short');
 if(env.API_TOKEN&&/^(.)\1+$/.test(env.API_TOKEN))failures.push('API_TOKEN cannot be a repeated character');
 try{
  const url=new URL(env.APP_PUBLIC_ORIGIN||'');
  if(url.protocol!=='https:'||url.username||url.password||url.pathname!=='/'||url.search||url.hash)throw Error();
 }catch{failures.push('APP_PUBLIC_ORIGIN must be HTTPS origin only')}
 try{
  const cfg=oidcConfig(env);
  if(cfg.adminGroup===cfg.viewerGroup)failures.push('admin and viewer group IDs must differ');
  if(cfg.redirect!==new URL('/api/auth/oidc/callback',env.APP_PUBLIC_ORIGIN).toString())failures.push('OIDC redirect must match public callback');
 }catch{failures.push('OIDC configuration invalid')}
 try{const uri=redisURLIsSafe(env);if(!uri?.startsWith('rediss://'))failures.push('verified TLS Redis required')}catch{failures.push('Redis configuration invalid')}
 if(!['true','false',undefined].includes(env.AI_ASSIST_ENABLED))failures.push('invalid AI_ASSIST_ENABLED');
 if(env.AI_ASSIST_ENABLED==='true'&&env.AI_DATA_SHARING_APPROVED!=='true')failures.push('AI provider sharing requires explicit approval');
 return failures;
}
export function productionErrors(env=process.env){
 if(env.ADB_PRODUCTION!=='true')return [];
 return check(env);
}
export function assertProduction(env=process.env){
 const errs=productionErrors(env);
 if(errs.length)throw Error('Production configuration refused: '+errs.join('; '));
}
