import { createHash, createHmac, randomBytes, timingSafeEqual } from 'node:crypto';
import { cookies } from 'next/headers';

export const COOKIE_NAME = 'adb_session';
const LIFE = 8 * 60 * 60;

function secret(): string {
  const key = process.env.APP_SESSION_SECRET;
  if (!key || key.length < 32) throw new Error('APP_SESSION_SECRET must be at least 32 characters');
  return key;
}
function signature(payload: string): string {
  return createHmac('sha256', secret()).update(payload).digest('base64url');
}
export function issueSession(now = Math.floor(Date.now() / 1000)): string {
  const payload = `${now + LIFE}.${randomBytes(16).toString('hex')}`;
  return `${payload}.${signature(payload)}`;
}
export function verifySession(session?: string): boolean {
  if (!session) return false;
  const parts = session.split('.');
  if (parts.length !== 3 || !/^\d{10}$/.test(parts[0]) || !/^[a-f0-9]{32}$/.test(parts[1])) return false;
  const expiry = Number(parts[0]);
  if (expiry <= Date.now() / 1000 || expiry > Date.now() / 1000 + LIFE + 60) return false;
  const expected = Buffer.from(signature(parts[0]+'.'+parts[1]), 'utf8');
  const got = Buffer.from(parts[2], 'utf8');
  return expected.length === got.length && timingSafeEqual(expected, got);
}
export async function authorized(): Promise<boolean> {
  return verifySession((await cookies()).get(COOKIE_NAME)?.value);
}
export function validPassword(submitted: string): boolean {
  const wanted = process.env.APP_PASSWORD;
  if (!wanted || wanted.length < 12) throw new Error('APP_PASSWORD must be at least 12 characters');
  const left = createHmac('sha256', 'adb-compare').update(submitted).digest();
  const right = createHmac('sha256', 'adb-compare').update(wanted).digest();
  return timingSafeEqual(left, right);
}
export function sameOrigin(request: Request): boolean {
  const origin = request.headers.get('origin');
  const host = request.headers.get('host');
  if (!origin || !host) return false;
  try { return new URL(origin).host === host; } catch { return false; }
}

// A distinct HMAC-proven owner scope for each signed browser session.
export function sessionScope(cookieValue:string,apiToken:string):{id:string;proof:string}{
 if(!verifySession(cookieValue))throw new Error('Invalid session');
 const id=createHash('sha256').update(cookieValue).digest('hex');
 const proof=createHmac('sha256',apiToken).update('adb-session:'+id).digest('hex');
 return {id,proof};
}
