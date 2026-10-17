export function totpValue(secret:string,timeMs?:number,digits?:number):string;
export function verifyTotp(code:string,secret:string,timeMs?:number):boolean;
export function totpEnabled():boolean;
