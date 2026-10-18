#!/usr/bin/env node
// Evidence-presence gate, NOT a security certification or source attestation.
import {readFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
const required=[
 'threatModelAndPenTest','liveSSOAndMFA','joinerMoverLeaver','redisFailover',
 'vaultKeyRotation','offHostAuditIntegrity','monitoringAlertDelivery',
 'performanceAndHA','disasterRecoveryRpoRto','rollbackExercise',
 'licenseAndSupplyChainReview','authorizedReleaseApproval'
];
export function evaluate(evidence,sha,now=new Date()){
 const failures=[];
 if(!/^[a-f0-9]{40}$/.test(sha))failures.push('a specific Git commit SHA is required');
 if(!evidence||typeof evidence!=='object'||Array.isArray(evidence))return ['evidence must be a JSON object'];
 if(evidence.commit!==sha)failures.push('evidence must match the exact commit being qualified');
 for(const key of required){
  const item=evidence[key];
  if(!item||typeof item!=='object'||Array.isArray(item)){failures.push(key+': external evidence is missing');continue}
  try{
   const url=new URL(item.url);
   if(url.protocol!=='https:'||url.username||url.password||url.hostname.endsWith('.example.com')||url.hostname.includes('example.invalid'))throw Error();
  }catch{failures.push(key+': HTTPS evidence URL required')}
  if(typeof item.reviewer!=='string'||item.reviewer.trim().length<3)failures.push(key+': reviewer identity missing');
  const at=Date.parse(item.reviewedAt);
  if(!Number.isFinite(at)||at>now.getTime()||now.getTime()-at>180*86400000)failures.push(key+': recent review date required');
  if(item.approved!==true)failures.push(key+': approval is not recorded');
 }
 return failures;
}
export function run(args=process.argv.slice(2)){
 const [path,sha]=args;
 if(!path||!sha)throw Error('usage: node check.mjs <evidence.json> <40-char-commit-sha>');
 let doc;try{doc=JSON.parse(readFileSync(path,'utf8'))}catch{throw Error('external qualification evidence file missing or invalid')}
 const failures=evaluate(doc,sha);
 if(failures.length)throw Error('Not qualified:\n'+failures.join('\n'));
 return 'Evidence file structurally complete. Human review and source authenticity must still be verified.';
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
 try{process.stdout.write(run()+'\n')}catch(err){process.stderr.write(err.message+'\n');process.exitCode=1}
}
