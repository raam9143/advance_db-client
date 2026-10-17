export function redisURLIsSafe(env?:NodeJS.ProcessEnv):string|null;
export function persistSession(cookie:string):Promise<boolean>;
export function sessionActive(cookie:string|undefined):Promise<boolean>;
export function revokeSession(cookie:string|undefined):Promise<boolean>;
export function consumeOTP(secret:string,otp:string):Promise<boolean>;
