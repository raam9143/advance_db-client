#!/usr/bin/env node
import { createInterface } from 'node:readline';
import { pathToFileURL } from 'node:url';

// Local-only read-only stdio bridge for MCP 2025-11-25 clients.
const VERSION='2025-11-25';
const ID_RE=/^[a-f0-9]{48}$/;
const TOOLS=[
{name:'connections_list',description:'List temporary database connections accessible to this API token',inputSchema:{type:'object',properties:{},additionalProperties:false}},
{name:'schema_list',description:'List tables in one connection',inputSchema:{type:'object',properties:{connectionId:{type:'string'}},required:['connectionId'],additionalProperties:false}},
{name:'sql_query_readonly',description:'Execute a read-only SQL query, capped at 1,000 rows',inputSchema:{type:'object',properties:{connectionId:{type:'string'},sql:{type:'string'},parameters:{type:'array',items:{type:['string','number','boolean','null']},maxItems:100}},required:['connectionId','sql'],additionalProperties:false}},
{name:'sql_explain',description:'Explain a SELECT or WITH query without executing writes',inputSchema:{type:'object',properties:{connectionId:{type:'string'},sql:{type:'string'},parameters:{type:'array',items:{type:['string','number','boolean','null']},maxItems:100}},required:['connectionId','sql'],additionalProperties:false}}
];
const error=(id,code,message)=>({jsonrpc:'2.0',id,error:{code,message}});
const result=(id,value)=>({jsonrpc:'2.0',id,result:value});
const toolError=message=>({content:[{type:'text',text:message}],isError:true});
function validateInput(name,args){
 if(!TOOLS.some(x=>x.name===name))throw Error('unknown tool');
 if(!args||typeof args!=='object'||Array.isArray(args))throw Error('invalid arguments');
 if(name!=='connections_list'&&!ID_RE.test(args.connectionId))throw Error('invalid connection ID');
 if(name.startsWith('sql_')){
  if(typeof args.sql!=='string'||args.sql.length>100000||!/^(SELECT|WITH)\s/i.test(args.sql.trim()))throw Error('tool accepts SELECT or WITH only');
  if(args.sql.includes(';')||args.sql.includes('--')||args.sql.includes('/*'))throw Error('comments and SQL statement separators are disallowed by MCP');
  if(args.parameters!==undefined&&(!Array.isArray(args.parameters)||args.parameters.length>100||args.parameters.some(p=>p!==null&&!['string','number','boolean'].includes(typeof p))))throw Error('invalid SQL parameters');
 }
}
async function backend(path,method,body,options){
 const base=options.base||process.env.MCP_GO_API_URL||'http://127.0.0.1:8080';
 const token=options.token??process.env.MCP_API_TOKEN;
 if(!token||!/^[\x21-\x7e]+$/.test(token))throw Error('MCP_API_TOKEN is required');
 const u=new URL(base);
 if(!['http:','https:'].includes(u.protocol)||u.username||u.password||u.pathname!=='/'||u.search||u.hash)throw Error('invalid MCP_GO_API_URL');
 const endpoint=new URL(path,u);
 const response=await (options.fetchImpl||fetch)(endpoint,{method,headers:{Authorization:`Bearer ${token}`,...(body?{'Content-Type':'application/json'}:{})},body:body?JSON.stringify(body):undefined,signal:AbortSignal.timeout(20000)});
 const data=await response.json();
 if(!response.ok)throw Error(typeof data.error==='string'?data.error:`backend HTTP ${response.status}`);
 return data;
}
export async function handle(message,options={}){
 if(!message||message.jsonrpc!=='2.0'||typeof message.method!=='string')return error(message?.id??null,-32600,'invalid request');
 if(message.id===undefined)return null;
 const id=message.id;
 try{
  switch(message.method){
   case 'initialize':return result(id,{protocolVersion:VERSION,serverInfo:{name:'advance-db-client-readonly',version:'0.1.0'},capabilities:{tools:{listChanged:false}}});
   case 'ping':return result(id,{});
   case 'tools/list':return result(id,{tools:TOOLS});
   case 'tools/call':{
    const name=message.params?.name,args=message.params?.arguments||{};
    try{
     validateInput(name,args);
     let path='/v1/connections',method='GET',body;
     if(name==='schema_list')path+=`/${args.connectionId}/schema`;
     if(name.startsWith('sql_')){
      path+=`/${args.connectionId}/${name==='sql_explain'?'explain':'query'}`;
      method='POST';body={sql:args.sql,parameters:args.parameters||[],timeoutSeconds:15};
     }
     const data=await backend(path,method,body,options);
     return result(id,{content:[{type:'text',text:JSON.stringify(data)}]});
    }catch(e){return result(id,toolError(e instanceof Error?e.message:'tool failed'))}
   }
   default:return error(id,-32601,'method not found');
  }
 }catch{return error(id,-32603,'internal error')}
}
async function stdio(){
 const rl=createInterface({input:process.stdin,crlfDelay:Infinity});
 for await(const line of rl){
  if(line.length>1_000_000){process.stdout.write(JSON.stringify(error(null,-32600,'message too large'))+'\n');continue}
  let request;try{request=JSON.parse(line)}catch{process.stdout.write(JSON.stringify(error(null,-32700,'parse error'))+'\n');continue}
  const response=await handle(request);
  if(response!==null)process.stdout.write(JSON.stringify(response)+'\n');
 }
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){void stdio()}
