import { test, expect } from '@playwright/test';
const password=process.env.APP_PASSWORD!;
const base='http://127.0.0.1:3000';
test('login, workspace and logout lifecycle',async({page})=>{
 await page.goto('/');
 await expect(page).toHaveURL(/\/login/);
 await expect(page.getByRole('link',{name:/Sign in with SSO/i})).toBeVisible();
 await page.locator('#password').fill(password);
 await page.getByRole('button',{name:/Sign in/i}).click();
 await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
 await page.getByRole('button',{name:'Log out'}).click();
 await expect(page).toHaveURL(/\/login/);
 await page.goto('/');
 await expect(page).toHaveURL(/\/login/);
});
test('cross-origin login is rejected',async({request})=>{
 const r=await request.post(base+'/api/auth/login',{data:{password},headers:{Origin:'https://example.invalid'}});
 expect(r.status()).toBe(403);
});
test('signed-in SQL read-only workspace through real Go API',async({page})=>{
 await page.goto('/login');
 await page.locator('#password').fill(password);
 await page.getByRole('button',{name:/Sign in/i}).click();
 await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
 const response=await page.request.post(base+'/api/backend/v1/connections',{
  headers:{Origin:base,'Content-Type':'application/json'},
  data:{name:'sqlite-readonly-e2e',engine:'sqlite',database:':memory:',readOnly:true}
 });
 expect(response.status(),await response.text()).toBe(201);
 const connection=await response.json();
 expect(connection.readOnly).toBe(true);
 const query=await page.request.post(base+`/api/backend/v1/connections/${connection.id}/query`,{
  headers:{Origin:base,'Content-Type':'application/json'},
  data:{sql:'SELECT 1 AS answer'}
 });
 expect(query.status(),await query.text()).toBe(200);
});
