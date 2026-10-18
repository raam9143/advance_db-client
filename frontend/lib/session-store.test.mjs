import test,{after} from 'node:test';
import assert from 'node:assert/strict';
import { randomBytes } from 'node:crypto';
import { redisURLIsSafe,persistSession,sessionActive,revokeSession,consumeOTP,closeSessionStore } from './session-store.mjs';

test('Redis session URL validation refuses remote plaintext',()=>{
 assert.throws(()=>redisURLIsSafe({SESSION_REDIS_URL:'redis://db.example.com:6379',SESSION_REDIS_ALLOW_PLAINTEXT_LOOPBACK:'true'}));
 assert.throws(()=>redisURLIsSafe({SESSION_REDIS_REQUIRED:'true'}));
 assert.equal(redisURLIsSafe({SESSION_REDIS_URL:'rediss://redis.example.com:6380'}),'rediss://redis.example.com:6380');
});
test('cross-process Redis backed revocation and single-use code',async(t)=>{
 if(!process.env.SESSION_REDIS_URL){t.skip('requires SESSION_REDIS_URL');return}
 const cookie='v2.'+(Math.floor(Date.now()/1000)+120)+'.'+randomBytes(16).toString('hex')+'.viewer.test.'+randomBytes(32).toString('base64url');
 assert.equal(await sessionActive(cookie),false);
 assert.equal(await persistSession(cookie),true);
 assert.equal(await sessionActive(cookie),true);
 assert.equal(await revokeSession(cookie),true);
 assert.equal(await sessionActive(cookie),false);
 const secret=randomBytes(32).toString('hex'),otp='400123';
 assert.equal(await consumeOTP(secret,otp),true);
 assert.equal(await consumeOTP(secret,otp),false);
});

after(async()=>{await closeSessionStore()});

test('distributed local-admin login lockout',async(t)=>{
 if(!process.env.SESSION_REDIS_URL){t.skip('requires SESSION_REDIS_URL');return}
 const {loginAllowed,recordLoginFailure,clearLoginFailures}=await import('./session-store.mjs');
 await clearLoginFailures();
 assert.equal(await loginAllowed(),true);
 for(let i=0;i<5;i++)await recordLoginFailure();
 assert.equal(await loginAllowed(),false);
 await clearLoginFailures();
 assert.equal(await loginAllowed(),true);
});
