import test from 'node:test';
import assert from 'node:assert/strict';
import {handle} from './server.mjs';
const mk=(method,params)=>({jsonrpc:'2.0',id:1,method,params});
const id='a'.repeat(48);
test('initialization and discoverable tools',async()=>{
 assert.equal((await handle(mk('initialize',{}))).result.protocolVersion,'2025-11-25');
 assert.equal((await handle(mk('tools/list'))).result.tools.length,4);
 assert.equal(await handle({jsonrpc:'2.0',method:'notifications/initialized'}),null);
});
test('rejects write queries before backend call',async()=>{
 let calls=0;
 const r=await handle(mk('tools/call',{name:'sql_query_readonly',arguments:{connectionId:id,sql:'DELETE FROM users'}}),{fetchImpl:()=>{calls++;return null}});
 assert.equal(calls,0);
 assert.equal(r.result.isError,true);
});
test('sends token and bounded SQL arguments only',async()=>{
 const fn=async(url,opts)=>{
  assert.equal(url.pathname,`/v1/connections/${id}/query`);
  assert.equal(opts.headers.Authorization,'Bearer test-secret');
  assert.equal(opts.method,'POST');
  assert.deepEqual(JSON.parse(opts.body).parameters,[4]);
  assert.equal(JSON.parse(opts.body).confirmWrite,undefined);
  return new Response(JSON.stringify({columns:['answer'],rows:[[4]]}),{status:200,headers:{'content-type':'application/json'}});
 };
 const r=await handle(mk('tools/call',{name:'sql_query_readonly',arguments:{connectionId:id,sql:'SELECT ?',parameters:[4]}}),{fetchImpl:fn,token:'test-secret'});
 assert.equal(r.result.isError,undefined);
 assert.match(r.result.content[0].text,/answer/);
});
test('requires API secret',async()=>{
 const r=await handle(mk('tools/call',{name:'connections_list',arguments:{}}),{token:''});
 assert.equal(r.result.isError,true);
});
