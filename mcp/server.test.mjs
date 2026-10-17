import test from 'node:test';
import assert from 'node:assert/strict';
import {handle} from './server.mjs';
const mk=(method,params)=>({jsonrpc:'2.0',id:1,method,params});
const id='a'.repeat(48);
test('initialization and discoverable tools',async()=>{
 assert.equal((await handle(mk('initialize',{}))).result.protocolVersion,'2025-11-25');
 assert.equal((await handle(mk('tools/list'))).result.tools.length,6);
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

test('MongoDB MCP rejects operator injection before backend call',async()=>{
 let invoked=0;
 const options={token:'token',fetchImpl:()=>{invoked++;throw Error('must not call')}};
 for(const unsafe of [
  {collection:'$cmd',limit:10,offset:0},
  {collection:'system.users',limit:10,offset:0},
  {collection:'safe',limit:101,offset:0},
  {collection:'safe',limit:10,offset:0,filterField:'$where',filterValue:'yes'},
  {collection:'safe',limit:10,offset:0,filterField:'user',filterValue:{$ne:'x'}},
 ]){
  const res=await handle(mk('tools/call',{name:'mongo_find_readonly',arguments:{connectionId:id,...unsafe}}),options);
  assert.equal(res.result.isError,true);
 }
 assert.equal(invoked,0);
});
test('MongoDB MCP forwards only scalar equality query',async()=>{
 const fetchImpl=async(url,opts)=>{
  assert.equal(url.pathname,`/v1/connections/${id}/mongo/find`);
  assert.equal(opts.method,'POST');
  const body=JSON.parse(opts.body);
  assert.deepEqual(body,{collection:'users',limit:10,offset:0,filterField:'status',filterValue:'active'});
  return new Response(JSON.stringify({collection:'users',documents:[{status:'active'}],nextOffset:null}),{status:200});
 };
 const res=await handle(mk('tools/call',{name:'mongo_find_readonly',arguments:{connectionId:id,collection:'users',limit:10,offset:0,filterField:'status',filterValue:'active'}}),{token:'test',fetchImpl});
 assert.equal(res.result.isError,undefined);
 assert.match(res.result.content[0].text,/active/);
});
