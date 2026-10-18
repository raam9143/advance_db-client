import { expect, test } from '@playwright/test';
import { mkdir } from 'node:fs/promises';

// All screenshots use disposable CI credentials and an empty SQL workspace.
// Do not capture real database results or authentication secrets in artifacts.
test('responsive visual review captures and horizontal-overflow guard',async({page})=>{
  const folder='test-results/ui-review';
  await mkdir(folder,{recursive:true});
  await page.setViewportSize({width:1440,height:900});
  await page.goto('/login');
  await expect(page.getByRole('heading',{name:'Sign in to your workspace'})).toBeVisible();
  await page.screenshot({path:folder+'/login-desktop.png',fullPage:true,animations:'disabled'});
  await page.setViewportSize({width:375,height:812});
  await page.screenshot({path:folder+'/login-mobile.png',fullPage:true,animations:'disabled'});
  await page.locator('#password').fill(process.env.APP_PASSWORD!);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
  await expect(page.locator('.shell')).toHaveAttribute('data-hydrated','true');
  for(const [label,width,height] of [
    ['desktop',1440,900],['tablet',768,1024],
    ['mobile',375,812],['small-mobile',320,800]
  ] as const){
    await page.setViewportSize({width,height});
    await page.reload();
    await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
  await expect(page.locator('.shell')).toHaveAttribute('data-hydrated','true');
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-window.innerWidth)).toBeLessThanOrEqual(2);
    await page.screenshot({path:folder+'/workspace-'+label+'.png',fullPage:true,animations:'disabled'});
  }
});
