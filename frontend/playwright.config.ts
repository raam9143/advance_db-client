import { defineConfig } from '@playwright/test';
export default defineConfig({
 testDir:'./tests',timeout:30000,expect:{timeout:15000},
 retries:process.env.CI?1:0,
 use:{baseURL:'http://127.0.0.1:3000',headless:true,trace:'retain-on-failure'},
 projects:[{name:'chromium',use:{browserName:'chromium'}}],
 webServer:[
  {command:'cd ../backend && go run ./cmd/api',url:'http://127.0.0.1:8080/health',reuseExistingServer:!process.env.CI,timeout:240000},
  {command:'npm run dev -- --hostname 127.0.0.1',url:'http://127.0.0.1:3000/login',reuseExistingServer:!process.env.CI,timeout:180000}
 ]
});
