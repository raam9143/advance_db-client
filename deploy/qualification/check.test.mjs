import test from 'node:test';
import assert from 'node:assert/strict';
import {evaluate} from './check.mjs';
const sha='a'.repeat(40);
const labels=['threatModelAndPenTest','liveSSOAndMFA','joinerMoverLeaver','redisFailover','vaultKeyRotation','offHostAuditIntegrity','monitoringAlertDelivery','performanceAndHA','disasterRecoveryRpoRto','rollbackExercise','licenseAndSupplyChainReview','authorizedReleaseApproval'];
function full(){
 const evidence={commit:sha};
 for(const k of labels)evidence[k]={url:'https://evidence.company.internal/report',reviewedAt:'2026-10-18T00:00:00Z',reviewer:'security-approver',approved:true};
 return evidence;
}
test('qualification gate never auto-approves absent real evidence',()=>{
 const empty=evaluate({commit:sha},sha,new Date('2026-10-18T10:00:00Z'));
 assert.equal(empty.length,labels.length);
 assert.ok(evaluate(full(),'b'.repeat(40),new Date('2026-10-18T10:00:00Z')).length>0);
});
test('rejects stale, placeholder, unreviewed, or HTTP evidence',()=>{
 const date=new Date('2026-10-18T10:00:00Z');
 for(const change of [
  {url:'http://example.com'}, {url:'https://evidence.example.com'},
  {reviewedAt:'2020-01-01'}, {reviewer:''},{approved:false}
 ]){
  const evidence=full();evidence[labels[0]]={...evidence[labels[0]],...change};
  assert.ok(evaluate(evidence,sha,date).length>0);
 }
 assert.deepEqual(evaluate(full(),sha,date),[]);
});
