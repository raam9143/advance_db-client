import test from 'node:test';
import assert from 'node:assert/strict';
import {productionErrors,assertProduction} from './production-policy.mjs';
const valid=()=>({
 ADB_PRODUCTION:'true',
 API_TOKEN:'a0b1c2d3e4f5g6h7i8j9k0l1m2n3o4p5',
 APP_SESSION_SECRET:'0123456789abcdef0123456789abcdef0123456789',
 APP_PUBLIC_ORIGIN:'https://db.example.com',
 OIDC_REQUIRED:'true',SESSION_REDIS_REQUIRED:'true',
 SESSION_REDIS_URL:'rediss://redis.example.com:6380',
 OIDC_ISSUER:'https://login.example.com/id',OIDC_CLIENT_ID:'adb-app',
 OIDC_REDIRECT_URI:'https://db.example.com/api/auth/oidc/callback',
 OIDC_ADMIN_GROUP:'db-admin',OIDC_VIEWER_GROUP:'db-viewer',
 OIDC_REQUIRED_AMR:'mfa'
});
test('valid production policy and dev compatibility',()=>{
 assert.deepEqual(productionErrors(valid()),[]);
 assert.deepEqual(productionErrors({}),[]);
 assert.doesNotThrow(()=>assertProduction(valid()));
});
test('production deployment rejects insecure authorization/networking',()=>{
 const changes=[
 {OIDC_REQUIRED:'false'}, {APP_PASSWORD:'something-sensitive'},
 {SESSION_REDIS_URL:'redis://redis.example.com:6379'}, {SESSION_REDIS_REQUIRED:'false'},
 {APP_PUBLIC_ORIGIN:'http://db.example.com'}, {OIDC_REQUIRED_AMR:''},
 {OIDC_ADMIN_GROUP:'db-viewer'}, {OIDC_REDIRECT_URI:'https://other.example.com/callback'},
 {APP_SESSION_SECRET:'short'}, {API_TOKEN:'weak'}, {SESSION_REDIS_ALLOW_PLAINTEXT_LOOPBACK:'true'},
 {AI_ASSIST_ENABLED:'true'}, {API_TOKEN:'x'.repeat(32)}
 ];
 for(const change of changes){
  const env={...valid(),...change};
  assert.ok(productionErrors(env).length>0,JSON.stringify(change));
  assert.throws(()=>assertProduction(env),/Production configuration refused/);
 }
});
