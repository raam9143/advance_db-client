'use client';
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Activity, ChevronDown, CircleCheck, Clock3, Code2, Database, Download, HardDrive, Layers, LoaderCircle, LogOut, PanelLeftClose, Play, Plus, RefreshCw, Search, Server, Shield, ShieldAlert, Table2, Trash2, X } from 'lucide-react';

type Engine='postgres'|'mysql'|'sqlite'|'mariadb'|'cockroach'|'clickhouse'|'redis'|'oracle'|'sqlserver'|'mongo';
type Connection={id:string;name:string;engine:Engine;database:string;host:string;readOnly:boolean};
type SchemaTable={schema:string;name:string};
type ColumnMeta={name:string;type:string;nullable:boolean;primaryKey:boolean;default:string|null};
type IndexMeta={name:string;column:string;unique:boolean};
type TableMeta={schema:string;name:string;columns:ColumnMeta[];indexes:IndexMeta[]};
type TablePage={columns:string[];rows:(string|number|boolean|null)[][];rowCount:number;truncated:boolean;limit:number;offset:number;nextOffset:number|null};
type QueryResult={columns:string[];rows:(string|number|boolean|null)[][];rowCount:number;truncated:boolean;durationMs:number;affected?:number};
type HistoryItem={sql:string;time:string;engine:string;success:boolean};
type SQLTab={id:number;label:string;sql:string};
const engines:{id:Engine;label:string;port:number;icon:string}[]=[{id:'postgres',label:'PostgreSQL',port:5432,icon:'P'},{id:'mysql',label:'MySQL',port:3306,icon:'M'},{id:'sqlite',label:'SQLite',port:0,icon:'S'},{id:'mariadb',label:'MariaDB',port:3306,icon:'M'},{id:'cockroach',label:'CockroachDB',port:26257,icon:'C'},{id:'clickhouse',label:'ClickHouse HTTPS (read-only)',port:8443,icon:'C'},{id:'redis',label:'Redis (read-only inspector)',port:6380,icon:'R'},{id:'sqlserver',label:'Microsoft SQL Server (TLS)',port:1433,icon:'S'},{id:'oracle',label:'Oracle Database (TCPS)',port:2484,icon:'O'},{id:'mongo',label:'MongoDB (read-only)',port:27017,icon:'M'}];
const initial={credentialRef:'',name:'',engine:'postgres' as Engine,host:'localhost',port:5432,database:'',user:'',password:'',sslMode:'require',readOnly:true,serviceMode:'service',tunnelEnabled:false,sshHost:'',sshPort:22,sshUser:'',sshPrivateKey:''};
const storage='adb.queryHistory.v1';
const savedStorage='adb.savedQueries.v1';
async function api<T>(path:string,method='GET',body?:unknown,signal?:AbortSignal):Promise<T>{const r=await fetch('/api/backend'+path,{method,headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined,cache:'no-store',signal});if(r.status===204)return undefined as T;const value=await r.json();if(!r.ok)throw Error(value.error||'Request failed');return value as T}
function fmt(value:unknown):string{if(value===null||value===undefined)return 'NULL';if(typeof value==='object')return JSON.stringify(value);return String(value)}
function sqlIdentifier(s:string,engine:Engine){if(engine==='sqlserver')return '['+s.replaceAll(']',']]')+']';const mark=(engine==='mysql'||engine==='mariadb')?'\x60':'"';return mark+s.replaceAll(mark,mark+mark)+mark}

export default function Workspace(){
 const pendingQuery=useRef<AbortController|null>(null);
 const [aiBusy,setAIBusy]=useState(false);
 const [aiAdvice,setAIAdvice]=useState('');
 const [backupBusy,setBackupBusy]=useState(false);
 const [mongoCollection,setMongoCollection]=useState('');
 const [mongoField,setMongoField]=useState('');
 const [mongoFilterValue,setMongoFilterValue]=useState('"active"');
 const [mongoLimit,setMongoLimit]=useState(25);
 const [mongoResult,setMongoResult]=useState<{collection:string;documents:unknown[];limit:number;offset:number;nextOffset:number|null}|null>(null);
 const [mongoBusy,setMongoBusy]=useState(false);
 const [redisCommand,setRedisCommand]=useState<'SCAN'|'GET'|'TYPE'|'TTL'|'HGETALL'|'LRANGE'>('SCAN');
 const [redisKey,setRedisKey]=useState('');const [redisPattern,setRedisPattern]=useState('*');const [redisCursor,setRedisCursor]=useState('0');
 const [redisResult,setRedisResult]=useState<{command:string;value:unknown}|null>(null);const [redisBusy,setRedisBusy]=useState(false);
 const [parameters,setParameters]=useState('[]');const [savedQueries,setSavedQueries]=useState<{name:string;sql:string}[]>([]);const [selectedTable,setSelectedTable]=useState<SchemaTable|null>(null);const [tableMeta,setTableMeta]=useState<TableMeta|null>(null);const [tablePage,setTablePage]=useState<TablePage|null>(null);const [tableBusy,setTableBusy]=useState(false);
 const [connections,setConnections]=useState<Connection[]>([]);const [activeId,setActiveId]=useState('');const [tables,setTables]=useState<SchemaTable[]>([]);const [query,setQueryValue]=useState('SELECT 1 AS hello_world;');
 const [queryTabs,setQueryTabs]=useState<SQLTab[]>([{id:1,label:'Query 1',sql:'SELECT 1 AS hello_world;'}]);
 const [activeQueryTab,setActiveQueryTab]=useState(1);const [result,setResult]=useState<QueryResult|null>(null);const [tab,setTab]=useState<'results'|'history'>('results');const [history,setHistory]=useState<HistoryItem[]>([]);const [busy,setBusy]=useState(false);const [connecting,setConnecting]=useState(false);const [modal,setModal]=useState(false);const [sidebar,setSidebar]=useState(true);const [error,setError]=useState('');const [filter,setFilter]=useState('');const [form,setForm]=useState(initial);
 function setQuery(next:string){
   setQueryValue(next);
   setQueryTabs(tabs=>tabs.map(t=>t.id===activeQueryTab?{...t,sql:next}:t));
 }
 function openQueryTab(){
   if(queryTabs.length>=8){setError('Maximum eight SQL tabs');return}
   const id=Math.max(0,...queryTabs.map(t=>t.id))+1;
   setQueryTabs(tabs=>[...tabs,{id,label:`Query ${id}`,sql:'SELECT 1;'}]);
   setActiveQueryTab(id);setQueryValue('SELECT 1;');setResult(null);
 }
 function switchQueryTab(id:number){
   const next=queryTabs.find(t=>t.id===id);
   if(!next||busy)return;
   setActiveQueryTab(id);setQueryValue(next.sql);setResult(null);setError('');
 }
 function closeQueryTab(id:number){
   if(queryTabs.length<=1||busy)return;
   const remaining=queryTabs.filter(t=>t.id!==id);
   setQueryTabs(remaining);
   if(id===activeQueryTab){
    const next=remaining[remaining.length-1];
    setActiveQueryTab(next.id);setQueryValue(next.sql);setResult(null);
   }
 }
 const active=connections.find(c=>c.id===activeId);
 const load=useCallback(async()=>{try{const c=await api<Connection[]>('/v1/connections');setConnections(c);setActiveId(previous=>c.some(a=>a.id===previous)?previous:(c[0]?.id||''))}catch(e){setError(e instanceof Error?e.message:'Could not load connections')}},[]);
 useEffect(()=>{void load();try{const v=JSON.parse(localStorage.getItem(storage)||'[]') as HistoryItem[];if(Array.isArray(v))setHistory(v.slice(0,25));const sq=JSON.parse(localStorage.getItem(savedStorage)||'[]') as {name:string;sql:string}[];if(Array.isArray(sq))setSavedQueries(sq.slice(0,50).filter(q=>typeof q.name==='string'&&typeof q.sql==='string'))}catch{/* invalid local history ignored */}},[load]);
 useEffect(()=>{if(!activeId||connections.find(c=>c.id===activeId)?.engine==='redis'){setTables([]);setSelectedTable(null);setTableMeta(null);setTablePage(null);return}let cancelled=false;api<SchemaTable[]>(`/v1/connections/${activeId}/schema`).then(x=>{if(!cancelled)setTables(x)}).catch(e=>{if(!cancelled){setTables([]);setError(e.message)}});return ()=>{cancelled=true}},[activeId,connections]);
 const filtered=useMemo(()=>tables.filter(t=>`${t.schema}.${t.name}`.toLowerCase().includes(filter.toLowerCase())),[tables,filter]);
 function changeEngine(value:Engine){const selected=engines.find(e=>e.id===value)!;setForm(f=>({...f,engine:value,port:selected.port,sslMode:value==='mysql'||value==='mariadb'?'true':'require',host:value==='sqlite'?'':f.host,readOnly:value==='redis'||value==='mongo'?true:f.readOnly,serviceMode:value==='oracle'?'service':'',tunnelEnabled:false}))}
 async function connect(e:FormEvent){e.preventDefault();setConnecting(true);setError('');try{const {credentialRef,tunnelEnabled,sshHost,sshPort,sshUser,sshPrivateKey,...dbConfig}=form;
  const payload=credentialRef.trim()?{name:form.name,credentialRef:credentialRef.trim(),readOnly:true}
   :{...dbConfig,...(tunnelEnabled?{tunnel:{host:sshHost,port:sshPort,user:sshUser,privateKey:sshPrivateKey}}:{})};
  const conn=await api<Connection>('/v1/connections','POST',payload);setConnections(v=>[...v,conn]);setActiveId(conn.id);setForm(initial);setModal(false);setResult(null)}catch(e){setError(e instanceof Error?e.message:'Connection failed')}finally{setConnecting(false)}}
 async function remove(id:string){if(!window.confirm('Disconnect this database session?'))return;try{await api<void>(`/v1/connections/${id}`,'DELETE');setConnections(v=>v.filter(c=>c.id!==id));if(activeId===id){setActiveId('');setResult(null);setSelectedTable(null);setTablePage(null);setTableMeta(null)}}catch(e){setError(e instanceof Error?e.message:'Disconnect failed')}}
 function pushHistory(item:HistoryItem){setHistory(prev=>{const next=[item,...prev].slice(0,25);try{localStorage.setItem(storage,JSON.stringify(next))}catch{}return next})}
 function parsedParameters(): unknown[] {const p:unknown=JSON.parse(parameters);if(!Array.isArray(p)||p.length>100||p.some(v=>v!==null&&!['string','number','boolean'].includes(typeof v)))throw Error('Parameters must be a JSON array (max 100) of strings, numbers, booleans or null');return p}
 function saveQuery(){const name=window.prompt('Name for saved SQL query');if(!name?.trim()||!query.trim())return;const next=[{name:name.trim().slice(0,80),sql:query},...savedQueries.filter(s=>s.name!==name.trim())].slice(0,50);setSavedQueries(next);try{localStorage.setItem(savedStorage,JSON.stringify(next))}catch{/* browser storage unavailable */}}
 async function execute(explain=false){if(!active||!query.trim()||busy)return;setError('');let args:unknown[];try{args=parsedParameters()}catch(e){setError(e instanceof Error?e.message:'Invalid parameters');return}
 const isRead=/^\s*(select|with|explain|show)\s/i.test(query);const confirmWrite=!explain&&!isRead;if(confirmWrite&&(active.readOnly||!window.confirm('This operation may modify data in '+active.name+'. Proceed with this exact SQL statement?')))return;
 const controller=new AbortController();pendingQuery.current=controller;setBusy(true);setTab('results');const now=new Date().toLocaleString();try{const data=await api<QueryResult>(`/v1/connections/${active.id}/${explain?'explain':'query'}`,'POST',{sql:query,parameters:args,timeoutSeconds:20,confirmWrite},controller.signal);setResult(data);if(!explain)pushHistory({sql:query,time:now,engine:active.engine,success:true})}catch(e){setResult(null);setError(controller.signal.aborted?'Query cancelled':e instanceof Error?e.message:'Query failed');if(!explain&&!controller.signal.aborted)pushHistory({sql:query,time:now,engine:active.engine,success:false})}finally{if(pendingQuery.current===controller)pendingQuery.current=null;setBusy(false)}}
 async function exportData(format:'csv'|'json'){if(!active||busy||!query.trim())return;setError('');try{const args=parsedParameters();const r=await fetch(`/api/backend/v1/connections/${active.id}/export?format=${format}`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({sql:query,parameters:args,timeoutSeconds:20})});if(!r.ok){const j=await r.json();throw Error(j.error||'Export failed')}const blob=await r.blob();const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download='results.'+format;a.click();URL.revokeObjectURL(url)}catch(e){setError(e instanceof Error?e.message:'Export failed')}}
 async function streamCSV(){
  if(!active||busy||!active.readOnly||!query.trim())return;
  if(!window.confirm('Export up to 50,000 rows as CSV? Sensitive database data will be downloaded to this device. The file may be truncated; verify the result before use.'))return;
  setBusy(true);setError('');
  try{
   const params=parsedParameters();
   const response=await fetch(`/api/backend/v1/connections/${active.id}/stream`,{
    method:'POST',headers:{'Content-Type':'application/json'},
    body:JSON.stringify({sql:query,parameters:params,timeoutSeconds:55}),
    signal:AbortSignal.timeout(70000)
   });
   if(!response.ok){const e=await response.json();throw Error(e.error||'CSV stream failed')}
   // Browser Blob is bounded by the 32 MiB server limit. The BFF streams the body.
   const blob=await response.blob();if(blob.size>32*1024*1024)throw Error('Export exceeded 32 MiB');
   const url=URL.createObjectURL(blob);
   const a=document.createElement('a');a.href=url;a.download='bounded-stream.csv';a.click();
   setTimeout(()=>URL.revokeObjectURL(url),1000);
  }catch(e){setError(e instanceof Error?e.message:'Streaming export failed')}
  finally{setBusy(false)}
 }
 async function reviewAI(){
  if(aiBusy||!query.trim())return;
  if(!window.confirm('Send a redacted copy of this read-only SQL statement to the configured external AI provider? SQL identifiers may reveal your schema. No database rows are sent.'))return;
  setError('');setAIBusy(true);setAIAdvice('');
  try{
   const result=await api<{advice:string;model:string;executed:boolean}>('/v1/ai/review','POST',{sql:query,purpose:'explain',consent:true});
   setAIAdvice(result.advice);
  }catch(e){setError(e instanceof Error?e.message:'AI review unavailable')}
  finally{setAIBusy(false)}
 }
 async function inspectMongo(offset=0){
  if(!active||active.engine!=='mongo'||mongoBusy)return;
  const collection=mongoCollection||tables[0]?.name;
  if(!collection){setError('Select a MongoDB collection');return}
  setError('');let value:unknown=undefined;
  if(mongoField.trim()){
   try{
    value=JSON.parse(mongoFilterValue);
    if(value!==null&&!['string','number','boolean'].includes(typeof value))throw Error('Expected scalar');
   }catch{setError('Filter value must be a valid JSON scalar, such as "active" or 42');return}
  }
  setMongoBusy(true);
  try{
   const data=await api<{collection:string;documents:unknown[];limit:number;offset:number;nextOffset:number|null}>(
    `/v1/connections/${active.id}/mongo/find`,'POST',{
    collection,limit:mongoLimit,offset,filterField:mongoField.trim(),
    ...(mongoField.trim()?{filterValue:value}:{})
   });
   setMongoResult(data);setMongoCollection(collection);
  }catch(e){setMongoResult(null);setError(e instanceof Error?e.message:'MongoDB query failed')}
  finally{setMongoBusy(false)}
 }
 async function inspectRedis(){
  if(!active||active.engine!=='redis'||redisBusy)return;
  setRedisBusy(true);setError('');
  try{
   const data=await api<{command:string;value:unknown}>(`/v1/connections/${active.id}/redis`,'POST',{
    command:redisCommand,key:redisKey,cursor:redisCursor,pattern:redisPattern,start:0,stop:49
   });
   setRedisResult(data);
   if(redisCommand==='SCAN'&&Array.isArray(data.value)&&typeof data.value[0]==='string')setRedisCursor(data.value[0]);
  }catch(e){setRedisResult(null);setError(e instanceof Error?e.message:'Redis inspect failed')}
  finally{setRedisBusy(false)}
 }
 async function backupSQLite(){
  if(!active||active.engine!=='sqlite'||backupBusy)return;
  if(!window.confirm('Create a temporary SQLite snapshot? This operation writes a private backup to the Go server and requires SQLITE_BACKUP_DIR.'))return;
  setBackupBusy(true);setError('');
  let backupId='';
  try{
   const data=await api<{backupId:string;downloadPath:string}>(
    `/v1/connections/${active.id}/backup`,'POST',{confirmBackup:true});
   backupId=data.backupId;
   const response=await fetch('/api/backend'+data.downloadPath,{cache:'no-store'});
   if(!response.ok)throw Error('Snapshot download failed');
   const blob=await response.blob(),url=URL.createObjectURL(blob);
   const anchor=document.createElement('a');anchor.href=url;anchor.download='advance-db-snapshot.sqlite';anchor.click();URL.revokeObjectURL(url);
  }catch(e){setError(e instanceof Error?e.message:'Snapshot failed')}
  finally{
   if(backupId){try{await api<void>(`/v1/backups/${backupId}`,'DELETE')}catch{/* expiry will clean it */}}
   setBackupBusy(false)
  }
 }
 async function logout(){await fetch('/api/auth/logout',{method:'POST'});window.location.replace('/login')}

 async function inspectTable(table:SchemaTable,offset=0){
  if(!active)return;
  setSelectedTable(table);setTableBusy(true);setError('');
  try{
   const params=`?schema=${encodeURIComponent(table.schema)}&table=${encodeURIComponent(table.name)}`;
   const [metadata,page]=await Promise.all([
    api<TableMeta>(`/v1/connections/${active.id}/describe${params}`),
    api<TablePage>(`/v1/connections/${active.id}/table${params}&limit=50&offset=${offset}`)]);
   setTableMeta(metadata);setTablePage(page)
  }catch(e){setTableMeta(null);setTablePage(null);setError(e instanceof Error?e.message:'Could not inspect table')}
  finally{setTableBusy(false)}
 }
 function selectTable(table:SchemaTable){if(!active)return;if(active.engine==='mongo'){setMongoCollection(table.name);setMongoResult(null);setSelectedTable(table);return}void inspectTable(table);const name=active.engine==='sqlite'?sqlIdentifier(table.name,active.engine):`${sqlIdentifier(table.schema,active.engine)}.${sqlIdentifier(table.name,active.engine)}`;setQuery(active.engine==='oracle'?`SELECT * FROM ${name} FETCH FIRST 100 ROWS ONLY;`:active.engine==='sqlserver'?`SELECT TOP (100) * FROM ${name};`:`SELECT * FROM ${name}\nLIMIT 100;`);setTab('results')}
 return <div className="shell">
 <aside className={`sidebar ${sidebar?'':'sidebar-collapsed'}`}>
  <div className="sidebar-top"><span className="app-logo"><Database size={22}/></span><span className="brand-name">Advance<span>DB</span><small>CLIENT WORKSPACE</small></span><button className="icon-button collapse" title="Collapse sidebar" onClick={()=>setSidebar(false)}><PanelLeftClose size={18}/></button></div>
  <div className="sidebar-section-head"><span>WORKSPACE</span><button className="icon-button" title="Add connection" onClick={()=>{setError('');setModal(true)}}><Plus size={17}/></button></div>
  <div className="side-nav"><div className="nav-item current"><Layers size={17}/> SQL Workspace <span className="nav-tag">PRO</span></div><div className="nav-item"><Activity size={17}/> Activity <span className="soon">soon</span></div></div>
  <div className="sidebar-section-head"><span>SAVED SQL <em>{savedQueries.length}</em></span></div><div className="saved-query-list">{savedQueries.map((q,i)=><button key={i} onClick={()=>{setQuery(q.sql);setTab('results')}} title={q.name}>{q.name}</button>)}</div>
  <div className="sidebar-section-head"><span>CONNECTIONS <em>{connections.length}</em></span><button title="Refresh connections" className="icon-button" onClick={()=>void load()}><RefreshCw size={15}/></button></div>
  <div className="connection-list">{connections.length===0?<div className="empty-connection"><HardDrive size={22}/><strong>No connections yet</strong><p>Add a database to begin exploring.</p><button onClick={()=>{setError('');setModal(true)}}>+ New connection</button></div>:connections.map(c=><div key={c.id} className={`connection ${c.id===activeId?'selected':''}`}><button className="connection-main" onClick={()=>{setActiveId(c.id);setResult(null)}}><span className={`database-letter ${c.engine}`}>{engines.find(x=>x.id===c.engine)?.icon}</span><span className="conn-meta"><strong>{c.name}</strong><small>{c.database||c.host}</small></span>{c.id===activeId&&<span className="connection-dot"/>}</button><button className="icon-button conn-delete" title={`Disconnect ${c.name}`} onClick={()=>void remove(c.id)}><X size={14}/></button></div>)}</div>
  <div className="sidebar-bottom"><span className="online-dot"/> API session-based <button title="Log out" onClick={()=>void logout()} className="icon-button"><LogOut size={17}/></button></div>
 </aside>
 <main className="workspace-main">
  <header className="topbar">{!sidebar&&<button className="icon-button" title="Expand sidebar" onClick={()=>setSidebar(true)}><Layers size={20}/></button>}<div className="breadcrumbs">Workspace <span>/</span> <strong>SQL Editor</strong></div><div className="top-right"><span className="secure-label"><Shield size={14}/> Protected session</span><div className="avatar">A</div></div></header>
  <div className="workspace-content"><div className="page-heading"><div><div className="eyebrow">DATABASE EXPLORER</div><h1>SQL Workspace</h1><p>Query, explore, and manage your databases in one place.</p></div><button className="secondary new-connection" onClick={()=>{setError('');setModal(true)}}><Plus size={17}/> New connection</button></div>
  {error&&<div role="alert" className="error-bar"><ShieldAlert size={18}/><span>{error}</span><button onClick={()=>setError('')} aria-label="Dismiss error"><X size={16}/></button></div>}
  {active?.engine==='mongo'&&<section className="panel redis-inspector" aria-label="MongoDB read-only collection inspector">
   <div className="panel-heading"><div className="panel-title"><Database size={18}/><strong>MongoDB document inspector</strong></div><span className="muted-tag">Read-only · bounded</span></div>
   <p>Browse documents using an optional exact-match scalar filter. All operators, pipelines, writes, and arbitrary commands are disabled.</p>
   <div className="redis-fields">
    <label>Collection<select value={mongoCollection||tables[0]?.name||''} onChange={e=>{setMongoCollection(e.target.value);setMongoResult(null)}}>
      {tables.map(t=><option key={t.name} value={t.name}>{t.name}</option>)}
    </select></label>
    <label>Match field (optional)<input value={mongoField} maxLength={120} placeholder="status" onChange={e=>setMongoField(e.target.value)}/></label>
    {mongoField&&<label>Equal to (JSON scalar)<input value={mongoFilterValue} spellCheck={false} maxLength={4096} onChange={e=>setMongoFilterValue(e.target.value)} placeholder='"active"'/></label>}
    <label>Page size<select value={mongoLimit} onChange={e=>setMongoLimit(Number(e.target.value))}>{[10,25,50,100].map(v=><option key={v} value={v}>{v}</option>)}</select></label>
    <button className="primary" disabled={mongoBusy||!(mongoCollection||tables[0]?.name)} onClick={()=>void inspectMongo(0)}>{mongoBusy?'Loading…':'Find documents'}</button>
   </div>
   {mongoResult&&<div className="redis-result" aria-live="polite"><strong>{mongoResult.collection}: {mongoResult.documents.length} documents (offset {mongoResult.offset})</strong>
    <button className="secondary" onClick={()=>{
      const blob=new Blob([JSON.stringify(mongoResult.documents,null,2)],{type:'application/json'});
      const url=URL.createObjectURL(blob);
      const a=document.createElement('a');a.href=url;a.download='mongo-'+mongoResult.collection.replace(/[^A-Za-z0-9_-]/g,'_')+'-page.json';a.click();
      setTimeout(()=>URL.revokeObjectURL(url),1000);
    }}><Download size={14}/> Download page (Extended JSON)</button>
    <pre>{JSON.stringify(mongoResult.documents,null,2)}</pre>
    <div className="action-row"><button className="secondary" disabled={mongoBusy||mongoResult.offset===0} onClick={()=>void inspectMongo(Math.max(0,mongoResult.offset-mongoResult.limit))}>Previous page</button>
    <button className="secondary" disabled={mongoBusy||mongoResult.nextOffset===null} onClick={()=>void inspectMongo(mongoResult.nextOffset??0)}>Next page</button></div>
   </div>}
  </section>}
  {active?.engine==='redis'&&<section className="panel redis-inspector"><div className="panel-heading"><div className="panel-title"><Database size={18}/><strong>Redis read-only inspector</strong></div><span className="muted-tag">RESP2 · TLS</span></div>
  <p>Inspect keys and values without enabling write or administrative commands. Results are bounded; use SCAN rather than KEYS.</p>
  <div className="redis-fields"><label>Operation<select value={redisCommand} onChange={e=>{setRedisCommand(e.target.value as typeof redisCommand);setRedisResult(null)}}>{(['SCAN','GET','TYPE','TTL','HGETALL','LRANGE'] as const).map(x=><option key={x} value={x}>{x}</option>)}</select></label>
  {redisCommand==='SCAN'?<><label>Key pattern<input value={redisPattern} onChange={e=>setRedisPattern(e.target.value)} maxLength={256}/></label><label>Cursor<input value={redisCursor} onChange={e=>setRedisCursor(e.target.value)} /></label></>:<label>Key<input aria-label="Redis key" value={redisKey} maxLength={1024} onChange={e=>setRedisKey(e.target.value)}/></label>}
  <button className="primary" disabled={redisBusy||(redisCommand!=='SCAN'&&!redisKey)} onClick={()=>void inspectRedis()}>{redisBusy?'Loading…':'Inspect'}</button></div>
  {redisResult&&<div className="redis-result" aria-live="polite"><strong>{redisResult.command} result</strong><pre>{JSON.stringify(redisResult.value,null,2)}</pre></div>}
  </section>}
  {aiAdvice&&active&&active.engine!=='redis'&&active.engine!=='mongo'&&<section className="panel ai-advice-panel" aria-label="Non-executable AI SQL advice"><div className="panel-heading"><strong>AI SQL review — advisory only</strong><button className="secondary" onClick={()=>setAIAdvice('')}>Dismiss</button></div><p>Generated advice can be incorrect. Nothing was executed or applied.</p><pre>{aiAdvice}</pre></section>}
  <div className={`editor-layout ${active?.engine==='redis'||active?.engine==='mongo'?'redis-engine-hidden':''}`}><div className="query-column"><section className="panel query-panel"><div className="panel-heading"><div className="panel-title"><Code2 size={18}/><strong>Query Editor</strong><span className="muted-tag">SQL</span></div><div className="query-target">{active?<><span className={`database-letter tiny ${active.engine}`}>{engines.find(x=>x.id===active.engine)?.icon}</span><span className="target-text">{active.name}</span><ChevronDown size={14}/></>:<span>No connection selected</span>}</div></div><div className="sql-tabs" role="tablist" aria-label="SQL query tabs">
  {queryTabs.map(item=><div className="sql-tab-item" key={item.id}><button type="button" role="tab" aria-selected={item.id===activeQueryTab} className={item.id===activeQueryTab?'sql-tab-active':''} disabled={busy} onClick={()=>switchQueryTab(item.id)}>{item.label}</button>{queryTabs.length>1&&<button type="button" className="sql-tab-close" aria-label={`Close ${item.label}`} disabled={busy} onClick={()=>closeQueryTab(item.id)}><X size={12}/></button>}</div>)}
  <button type="button" className="sql-tab-new" onClick={openQueryTab} disabled={queryTabs.length>=8||busy} aria-label="New SQL tab"><Plus size={14}/> New tab</button>
 </div><div className="editor-toolbar"><span className="editor-badge"><span className="green-dot"/>{active?.engine?.toUpperCase()||'SQL'}</span><span className="toolbar-tip">⌘ / Ctrl + Enter to run</span></div><textarea spellCheck={false} aria-label="SQL query editor" className="sql-editor" value={query} onKeyDown={e=>{if(e.key==='Enter'&&(e.metaKey||e.ctrlKey)){e.preventDefault();void execute()}}} onChange={e=>setQuery(e.target.value)}/><div className="query-parameters"><label htmlFor="query-parameters">Bind parameters (JSON array; PostgreSQL $1, SQL Server @p1, Oracle :1, MySQL/SQLite ?, ClickHouse typed parameters)</label><input id="query-parameters" spellCheck={false} value={parameters} onChange={e=>setParameters(e.target.value)} placeholder='[42, "name"]'/></div><div className="query-footer"><span className="query-warning">{active?.readOnly?<><Shield size={14}/> Read-only session</>:active?<><ShieldAlert size={14}/> Write enabled</>:<><Server size={14}/> Connect a database to execute</>}</span><div className="action-row"><button disabled={!query.trim()} className="secondary" onClick={saveQuery}>Save SQL</button><button disabled={!active||aiBusy||!query.trim()} className="secondary" onClick={()=>void reviewAI()}>{aiBusy?'Reviewing…':'AI review (opt-in)'}</button>{active?.engine==='sqlite'&&<button disabled={busy||backupBusy} className="secondary" onClick={()=>void backupSQLite()}><Download size={15}/>{backupBusy?'Snapshotting…':'SQLite snapshot'}</button>}<button disabled={!active||busy} className="secondary" onClick={()=>void exportData('csv')}><Download size={15}/> CSV</button><button disabled={!active||busy} className="secondary" onClick={()=>void exportData('json')}><Download size={15}/> JSON</button>{active?.readOnly&&<button disabled={busy||!query.trim()} className="secondary" title="Up to 50,000 rows and 32 MiB; export may be truncated" onClick={()=>void streamCSV()}><Download size={15}/> Stream CSV</button>}<button disabled={!active||busy||(active?.engine==='clickhouse'||active?.engine==='oracle'||active?.engine==='sqlserver')} className="secondary" onClick={()=>void execute(true)}>Explain</button>{busy&&<button className="secondary" onClick={()=>pendingQuery.current?.abort()}>Cancel query</button>}<button disabled={!active||busy||!query.trim()} className="primary" onClick={()=>void execute()}>{busy?<LoaderCircle size={16} className="spin"/>:<Play size={15} fill="currentColor"/>}{busy?'Running…':'Run query'}</button></div></div></section>
  <section className="panel result-panel"><div className="results-head"><div className="tab-buttons"><button className={tab==='results'?'tab active':'tab'} onClick={()=>setTab('results')}><Table2 size={16}/> Results</button><button className={tab==='history'?'tab active':'tab'} onClick={()=>setTab('history')}><Clock3 size={16}/> History</button></div>{result&&tab==='results'&&<span className="result-count"><CircleCheck size={14}/> {result.affected!==undefined?`${result.affected} affected`: `${result.rowCount} rows`} <span>· {result.durationMs} ms</span></span>}</div>
   {tab==='results'?result?result.columns.length===0?<div className="result-empty"><CircleCheck size={30}/><h3>Statement executed</h3><p>{result.affected??0} row(s) affected in {result.durationMs} ms.</p></div>:<div className="table-wrap"><table className="results-table"><thead><tr><th className="rownum">#</th>{result.columns.map((c,i)=><th key={i}>{c}</th>)}</tr></thead><tbody>{result.rows.map((r,ri)=><tr key={ri}><td className="rownum">{ri+1}</td>{r.map((v,ci)=><td key={ci} className={v===null?'null':''} title={fmt(v)}>{fmt(v)}</td>)}</tr>)}</tbody></table>{result.truncated&&<div className="truncated">Results capped at 1,000 rows; add LIMIT for smaller queries.</div>}</div>:<div className="result-empty"><Table2 size={32}/><h3>Ready for your first query</h3><p>Run a SQL statement to see results here.</p></div>:<div className="history"><p className="history-note">Last 25 statements, stored only in this browser.</p>{history.length===0?<div className="result-empty compact"><Clock3 size={28}/><p>No query history yet.</p></div>:history.map((h,i)=><button className="history-entry" key={i} onClick={()=>{setQuery(h.sql);setTab('results')}}><span className={h.success?'history-ok':'history-error'}>●</span><code>{h.sql.slice(0,130)}</code><small>{h.time}</small></button>)}</div>}
  </section></div>
  <aside className="schema-panel panel"><div className="schema-heading"><div className="panel-title"><Database size={17}/><strong>Schema Explorer</strong></div><span className="schema-count">{tables.length}</span></div><div className="schema-search"><Search size={16}/><input placeholder="Search tables…" aria-label="Search database tables" value={filter} onChange={e=>setFilter(e.target.value)}/></div><div className="schema-list">{!active?<div className="schema-placeholder"><Database size={28}/><p>Select a connection to explore tables.</p></div>:filtered.length===0?<div className="schema-placeholder"><Search size={26}/><p>No tables found.</p></div>:filtered.map((t,i)=><button key={`${t.schema}.${t.name}.${i}`} className="schema-table" title={`Query ${t.schema}.${t.name}`} onClick={()=>selectTable(t)}><Table2 size={16}/><span><strong>{t.name}</strong><small>{t.schema}</small></span><ChevronDown size={14} className="table-chevron"/></button>)}</div>{selectedTable&&<div className="table-inspector" aria-live="polite">
  <div className="inspector-title"><strong>{selectedTable.name}</strong><span>{tableBusy?'Loading…':`${tableMeta?.columns.length??0} columns`}</span></div>
  {tableMeta&&<div className="inspector-columns">{tableMeta.columns.map(col=><div className="inspector-col" key={col.name}><span>{col.primaryKey?'🔑 ':''}{col.name}</span><small>{col.type}{col.nullable?' · NULL':''}</small></div>)}</div>}
  {tablePage&&<><div className="inspector-title"><strong>Data preview</strong><span>{tablePage.rowCount} rows</span></div>
   <div className="inspector-preview">{tablePage.rows.slice(0,8).map((row,i)=><div className="inspector-row" key={i}>{row.slice(0,3).map((v,j)=><span key={j} title={fmt(v)}>{fmt(v)}</span>)}</div>)}</div>
   <div className="inspector-pager"><button disabled={tableBusy||tablePage.offset===0} onClick={()=>void inspectTable(selectedTable,Math.max(0,tablePage.offset-50))}>Previous</button><small>{tablePage.offset+1}–{tablePage.offset+tablePage.rowCount}</small><button disabled={tableBusy||tablePage.nextOffset===null} onClick={()=>void inspectTable(selectedTable,tablePage.nextOffset??0)}>Next</button></div>
  </>}
 </div>}
 <div className="schema-footer">Click a table to preview rows and build a SELECT query</div></aside></div>
  <div className="workspace-foot"><div><span className="online-dot"/> Workspace ready</div><span>Go API <span className="dot-separator">•</span> Next.js / TypeScript <span className="dot-separator">•</span> Query timeout 30s</span></div>
  </div>
 </main>
 {modal&&<div className="modal-overlay" role="presentation" onMouseDown={e=>{if(e.target===e.currentTarget)setModal(false)}}><form className="modal" onSubmit={e=>void connect(e)}><div className="modal-top"><div><p className="eyebrow">CONNECTION SETTINGS</p><h2>New database connection</h2><p className="muted">Credentials stay on the Go server for this session only.</p></div><button type="button" aria-label="Close" className="icon-button" onClick={()=>setModal(false)}><X size={20}/></button></div><label className="field-label">Connection name<input required maxLength={120} placeholder="e.g. Production analytics" value={form.name} onChange={e=>setForm(f=>({...f,name:e.target.value}))}/></label><label className="field-label span2">Encrypted credential reference (admin only, optional)<input value={form.credentialRef} maxLength={64} placeholder="e.g. reporting_prod" onChange={e=>setForm(f=>({...f,credentialRef:e.target.value}))}/><small>When provided, the backend loads a pre-provisioned encrypted profile. No password is submitted from this form.</small></label><label className="field-label">Database engine<select disabled={!!form.credentialRef} value={form.engine} onChange={e=>changeEngine(e.target.value as Engine)}>{engines.map(e=><option key={e.id} value={e.id}>{e.label}</option>)}</select></label><div className="form-grid">{!form.credentialRef&&form.engine!=='sqlite'&&<><label className="field-label">Hostname<input required value={form.host} onChange={e=>setForm(f=>({...f,host:e.target.value}))}/></label><label className="field-label">Port<input required type="number" min={1} max={65535} value={form.port} onChange={e=>setForm(f=>({...f,port:Number(e.target.value)}))}/></label></>}{!form.credentialRef&&<label className="field-label span2">{form.engine==='sqlite'?'SQLite file path':form.engine==='redis'?'Database index (0–15)':form.engine==='oracle'?form.serviceMode==='sid'?'Oracle SID':'Oracle Service Name':'Database name'}<input required value={form.database} onChange={e=>setForm(f=>({...f,database:e.target.value}))} placeholder={form.engine==='sqlite'?'/data/app.db':form.engine==='redis'?'0':form.engine==='oracle'?'ORCLPDB1':form.engine==='mongo'?'app':'analytics'}/></label>}{!form.credentialRef&&form.engine==='oracle'&&<label className="field-label span2">Oracle connection identifier<select value={form.serviceMode} onChange={e=>setForm(f=>({...f,serviceMode:e.target.value}))}><option value="service">Service Name (PDB/service)</option><option value="sid">SID (legacy instance)</option></select></label>}{!form.credentialRef&&form.engine!=='sqlite'&&<><label className="field-label">Username<input required={form.engine!=='redis'&&form.engine!=='mongo'} autoComplete="off" value={form.user} onChange={e=>setForm(f=>({...f,user:e.target.value}))}/></label><label className="field-label">Password<input type="password" autoComplete="new-password" value={form.password} onChange={e=>setForm(f=>({...f,password:e.target.value}))}/></label><label className="field-label span2">TLS / SSL mode<select value={form.sslMode} onChange={e=>setForm(f=>({...f,sslMode:e.target.value}))}>{(form.engine==='postgres'||form.engine==='cockroach'||form.engine==='clickhouse'||form.engine==='redis'||form.engine==='oracle'||form.engine==='sqlserver'||form.engine==='mongo')?<><option value="require">Require TLS</option><option value="verify-full">Verify full certificate</option>{form.engine==='mongo'&&<option value="disable">Plaintext localhost (development only)</option>}{form.engine!=='clickhouse'&&form.engine!=='redis'&&form.engine!=='oracle'&&form.engine!=='sqlserver'&&form.engine!=='mongo'&&<><option value="verify-ca">Verify certificate authority</option><option value="disable">Disable TLS (local only)</option></>}</>:<><option value="true">Enable TLS</option><option value="false">Disable TLS (local only)</option></>}</select></label></>}</div>{!form.credentialRef&&(['postgres','cockroach','mysql','mariadb'] as Engine[]).includes(form.engine)&&<section className="ssh-connection" aria-label="SSH tunnel configuration">
 <label className="checkbox-row"><input type="checkbox" checked={form.tunnelEnabled} onChange={e=>setForm(f=>({...f,tunnelEnabled:e.target.checked,host:e.target.checked?'127.0.0.1':f.host,sslMode:e.target.checked?(f.engine==='mysql'||f.engine==='mariadb'?'false':'disable'):(f.engine==='mysql'||f.engine==='mariadb'?'true':'require')}))}/><span><strong>Use verified SSH tunnel</strong><small>Database must listen on the SSH server's loopback. Disabled unless the Go server's bastion, destination, and known-hosts allowlists are configured.</small></span></label>
 {form.tunnelEnabled&&<div className="form-grid">
 <label className="field-label">SSH bastion hostname<input required value={form.sshHost} onChange={e=>setForm(f=>({...f,sshHost:e.target.value}))} placeholder="bastion.internal"/></label>
 <label className="field-label">SSH port<input required type="number" min={1} max={65535} value={form.sshPort} onChange={e=>setForm(f=>({...f,sshPort:Number(e.target.value)}))}/></label>
 <label className="field-label span2">SSH username<input required value={form.sshUser} autoComplete="off" onChange={e=>setForm(f=>({...f,sshUser:e.target.value}))}/></label>
 <label className="field-label span2">SSH private key (PEM)<textarea required rows={5} spellCheck={false} autoComplete="off" maxLength={65536} value={form.sshPrivateKey} onChange={e=>setForm(f=>({...f,sshPrivateKey:e.target.value}))} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"/></label>
 <p className="field-label span2">Host keys are verified against a server-managed known_hosts file. The key is used for this session only; it is not saved in the browser.</p>
 </div>}
 </section>}<label className="checkbox-row"><input type="checkbox" disabled={!!form.credentialRef||form.engine==='redis'||form.engine==='clickhouse'||form.engine==='mongo'} checked={!!form.credentialRef||form.engine==='redis'||form.engine==='clickhouse'||form.readOnly} onChange={e=>setForm(f=>({...f,readOnly:e.target.checked}))}/><span><strong>Read-only session (recommended)</strong><small>Prevent common write statements. Use a DB read-only account for enforcement.</small></span></label>{error&&<div role="alert" className="modal-error">{error}</div>}<div className="modal-actions"><button type="button" className="secondary" onClick={()=>setModal(false)}>Cancel</button><button className="primary" type="submit" disabled={connecting}>{connecting?'Connecting…':'Connect database'}</button></div></form></div>}
 </div>
}
