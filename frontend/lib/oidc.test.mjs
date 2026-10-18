import test from 'node:test';
import assert from 'node:assert/strict';
import {generateKeyPairSync,createSign,createHash} from 'node:crypto';
import {beginFlow,verifyFlow,oidcConfig,authorizeURL,checkIDToken} from './oidc.mjs';
const config=oidcConfig({
 OIDC_ISSUER:'https://login.example.com/id',OIDC_CLIENT_ID:'advance-db',OIDC_REDIRECT_URI:'https://db.example.com/api/auth/oidc/callback',OIDC_ADMIN_GROUP:'db-admins',OIDC_VIEWER_GROUP:'db-readers',APP_SESSION_SECRET:'a secure but test-only 48 character HMAC key !!!!!'
});
function token(claimOverrides={},headerOverrides={}){
 const pair=generateKeyPairSync('rsa',{modulusLength:2048});
 const jwk=pair.publicKey.export({format:'jwk'});jwk.kid='test-key';
 const now=Math.floor(Date.now()/1000);
 const claims={iss:config.issuer,sub:'alice-123',aud:config.id,exp:now+300,iat:now,nonce:'test-nonce',groups:['db-readers'],...claimOverrides};
 const header={alg:'RS256',typ:'JWT',kid:'test-key',...headerOverrides};
 const input=Buffer.from(JSON.stringify(header)).toString('base64url')+'.'+Buffer.from(JSON.stringify(claims)).toString('base64url');
 const signer=createSign('RSA-SHA256');signer.update(input);signer.end();
 return {jwt:input+'.'+signer.sign(pair.privateKey).toString('base64url'),jwks:{keys:[jwk]}};
}
test('OIDC signed flow binds state, expires, and supplies PKCE',()=>{
 const {flow,cookie}=beginFlow(config,Date.now());
 assert.equal(verifyFlow(config,cookie,flow.state).nonce,flow.nonce);
 assert.throws(()=>verifyFlow(config,cookie,flow.state+'x'));
 assert.throws(()=>verifyFlow(config,cookie,flow.state,Date.now()+400000));
 const url=new URL(authorizeURL(config,{authorization_endpoint:'https://login.example.com/id/authorize'},flow));
 assert.equal(url.searchParams.get('code_challenge_method'),'S256');
 assert.equal(url.searchParams.get('code_challenge'),createHash('sha256').update(flow.verifier).digest('base64url'));
});
test('OIDC cryptographically validates ID token and group roles',()=>{
 const {jwt,jwks}=token();
 const result=checkIDToken(jwt,config,'test-nonce',jwks);
 assert.equal(result.role,'viewer');
 assert.match(result.principal,/^[a-f0-9]{64}$/);
 const admin=token({groups:['db-admins']});
 assert.equal(checkIDToken(admin.jwt,config,'test-nonce',admin.jwks).role,'admin');
 for(const claims of [
  {nonce:'bad'},{iss:'https://evil.example.com'}, {aud:'other-client'},
  {exp:Math.floor(Date.now()/1000)-5}, {groups:['not-authorized']}
 ]){
  const bad=token(claims);
  assert.throws(()=>checkIDToken(bad.jwt,config,'test-nonce',bad.jwks));
 }
 const modified=jwt.slice(0,-4)+'abcd';
 assert.throws(()=>checkIDToken(modified,config,'test-nonce',jwks));
 const none=token({}, {alg:'none'});
 assert.throws(()=>checkIDToken(none.jwt,config,'test-nonce',none.jwks));
});
test('OIDC configuration rejects HTTP or missing group mapping',()=>{
 assert.throws(()=>oidcConfig({...process.env,OIDC_ISSUER:'http://insecure.example.com',OIDC_CLIENT_ID:'x',OIDC_REDIRECT_URI:'https://db.example.com/callback',OIDC_ADMIN_GROUP:'a',OIDC_VIEWER_GROUP:'v',APP_SESSION_SECRET:'a'.repeat(40)}));
});

test('OIDC prevents multi-audience token substitution and requires IdP MFA',()=>{
 const mfa={...config,requiredAmr:'mfa'};
 const pass=token({amr:['pwd','mfa'],aud:[config.id,'another-client'],azp:config.id});
 assert.equal(checkIDToken(pass.jwt,mfa,'test-nonce',pass.jwks).role,'viewer');
 for(const claims of [
  {amr:['pwd']},{amr:[]},
  {amr:['mfa'],aud:[config.id,'attacker'],azp:'attacker'},
  {amr:['mfa'],aud:[config.id,'attacker']}
 ]){
  const bad=token(claims);
  assert.throws(()=>checkIDToken(bad.jwt,mfa,'test-nonce',bad.jwks));
 }
});
