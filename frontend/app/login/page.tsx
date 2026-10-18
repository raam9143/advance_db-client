'use client';
import { FormEvent, useState } from 'react';
import { ArrowRight, Database, LockKeyhole, Layers3, ShieldCheck } from 'lucide-react';

export default function Login(){
 const [password,setPassword]=useState('');
 const [otp,setOtp]=useState('');
 const [error,setError]=useState('');
 const [busy,setBusy]=useState(false);

 async function submit(e:FormEvent){
   e.preventDefault();setBusy(true);setError('');
   try{
     const response=await fetch('/api/auth/login',{
       method:'POST',headers:{'Content-Type':'application/json'},
       body:JSON.stringify({password,otp})
     });
     if(!response.ok){const body=await response.json();throw Error(body.error||'Login failed')}
     window.location.replace('/');
   }catch(e){setError(e instanceof Error?e.message:'Login failed')}
   finally{setBusy(false)}
 }

 return <main className="login-bg">
   <section className="login-showcase" aria-label="About Advance DB Client">
     <div className="login-brand">
       <span className="app-logo" aria-hidden="true"><Database size={24}/></span>
       <span>Advance DB <small>DATA WORKSPACE</small></span>
     </div>
     <div className="login-showcase-center">
       <div className="login-showcase-label">ONE WORKSPACE. MULTIPLE DATABASES.</div>
       <h2>Your data tools.<br/><span>One clear workspace.</span></h2>
       <p>Explore database structures, inspect data and work with SQL in a focused, security-conscious environment.</p>
       <div className="login-features">
         <div className="login-feature"><Database size={20}/><span>Connect across supported database engines</span></div>
         <div className="login-feature"><Layers3 size={20}/><span>Inspect schemas, queries and result sets</span></div>
         <div className="login-feature"><ShieldCheck size={20}/><span>Read-only controls and session security</span></div>
       </div>
     </div>
     <div className="login-showcase-foot">Advance DB Client · Secure database workspace</div>
   </section>
   <div className="login-form-area">
     <form className="login-card" onSubmit={submit} aria-labelledby="sign-in-title">
       <p className="eyebrow"><span className="eyebrow-dot"/> WELCOME BACK</p>
       <h1 id="sign-in-title">Sign in to your workspace</h1>
       <p className="muted">Access your connected databases and SQL tools.</p>
       <a className="secondary sso-link" href="/api/auth/oidc/start"><ShieldCheck size={16}/> Sign in with SSO (OIDC) <ArrowRight size={15}/></a>
       <div className="login-divider">or use access credentials</div>
       <label className="field-label" htmlFor="password">Access password
         <input id="password" type="password" autoComplete="current-password" required minLength={12}
           value={password} onChange={e=>setPassword(e.target.value)} placeholder="Enter your access password"/>
       </label>
       <label className="field-label" htmlFor="otp">Authenticator code <span className="muted">(if enabled)</span>
         <input id="otp" type="text" inputMode="numeric" autoComplete="one-time-code"
           maxLength={6} pattern="[0-9]{6}" value={otp} onChange={e=>setOtp(e.target.value.replace(/[^0-9]/g,''))}
           placeholder="6-digit code"/>
       </label>
       {error&&<p role="alert" className="error">{error}</p>}
       <button className="primary full" type="submit" disabled={busy} aria-busy={busy}>
         {busy?'Signing in…':<>Sign in <ArrowRight size={17}/></>}
       </button>
       <p className="footnote"><span className="login-lock"><LockKeyhole size={13}/> Protected session</span><br/>Use HTTPS when accessing this service outside localhost.</p>
     </form>
   </div>
 </main>;
}
