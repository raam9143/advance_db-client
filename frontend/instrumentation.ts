import {assertProduction} from './lib/production-policy.mjs';
export async function register(){
 if(process.env.NEXT_RUNTIME==='nodejs')assertProduction(process.env);
}
