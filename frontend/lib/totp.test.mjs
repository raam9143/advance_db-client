import test from 'node:test';
import assert from 'node:assert/strict';
import {totpValue,verifyTotp} from './totp.mjs';
const secret='GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ';
test('RFC6238 SHA1 vector (6 and 8 digits)',()=>{
 assert.equal(totpValue(secret,59000,8),'94287082');
 assert.equal(totpValue(secret,59000,6),'287082');
 assert.equal(totpValue(secret,1111111109000,8),'07081804');
});
test('OTP rejects invalid values and replays',()=>{
 const now=1600000000000;
 assert.equal(verifyTotp('12345',secret,now),false);
 const code=totpValue(secret,now);
 assert.equal(verifyTotp(code,secret,now),true);
 assert.equal(verifyTotp(code,secret,now),false);
 assert.equal(verifyTotp('abcdef',secret,now),false);
});
test('MFA fails closed with invalid secret',()=>{
 assert.throws(()=>totpValue('bad',59000),/Base32|20 bytes/);
});
