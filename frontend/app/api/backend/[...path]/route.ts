import { COOKIE_NAME, authorized, sameOrigin, sessionScope } from '@/lib/auth';
import { cookies } from 'next/headers';
export const runtime='nodejs';
export const dynamic='force-dynamic';
const reply=(message:string,status:number)=>Response.json({error:message},{status});
async function proxy(req:Request, {params}:{params:Promise<{path:string[]}>}):Promise<Response>{
 if (!(await authorized())) return reply('Session expired',401);
 if (!['GET','POST','DELETE'].includes(req.method)) return reply('Method not allowed',405);
 if (req.method!=='GET' && !sameOrigin(req)) return reply('Invalid origin',403);
 const segments=(await params).path;
 const path='/'+segments.join('/');
 const allowed=path==='/v1/schema/diff' && req.method==='POST'
  || path==='/v1/capabilities' && req.method==='GET'
  || path==='/v1/connections' && ['GET','POST'].includes(req.method)
  || /^\/v1\/connections\/[a-f0-9]{48}$/.test(path)&&req.method==='DELETE'
  || /^\/v1\/connections\/[a-f0-9]{48}\/(schema|describe|table)$/.test(path)&&req.method==='GET'
  || /^\/v1\/connections\/[a-f0-9]{48}\/(query|export|explain)$/.test(path)&&req.method==='POST';
 if (!allowed) return reply('Not found',404);
 const token=process.env.API_TOKEN,base=process.env.GO_API_URL||'http://127.0.0.1:8080';
 if (!token || !base.startsWith('http')) return reply('API not configured',503);
 const input=req.method==='POST' ? await req.text() : undefined;
 if (input && input.length>1_000_000) return reply('Request too large',413);
 try {
  const scope=sessionScope((await cookies()).get(COOKIE_NAME)!.value,token);
  const upstream=await fetch(`${base}${path}${new URL(req.url).search}`,{method:req.method,headers:{Authorization:`Bearer ${token}`,'X-ADB-Session-ID':scope.id,'X-ADB-Session-Proof':scope.proof, ...(input?{'Content-Type':'application/json'}:{})},body:input,cache:'no-store',signal:AbortSignal.timeout(33000)});
  const format=new URL(req.url).searchParams.get('format');
  const isExport=path.endsWith('/export')&&upstream.ok;
  const isCSV=isExport&&format!=='json';
  return new Response(upstream.body,{status:upstream.status,headers:{'Content-Type':isCSV?'text/csv; charset=utf-8':'application/json','Cache-Control':'no-store',...(isExport?{'Content-Disposition':`attachment; filename="results.${isCSV?'csv':'json'}"`}:{})}});
 } catch {return reply('Database service unavailable',502)}
}
export {proxy as GET,proxy as POST,proxy as DELETE};
