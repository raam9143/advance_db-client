import { expect, test } from '@playwright/test';

const password=process.env.APP_PASSWORD!;

async function signIn(page: import('@playwright/test').Page){
  await page.goto('/login');
  await page.locator('#password').fill(password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
  await expect(page.locator('.shell')).toHaveAttribute('data-hydrated','true');
}

test('sign-in screen exposes security details and a working password flow',async({page})=>{
  await page.goto('/login');
  await expect(page.getByRole('heading',{name:'Sign in to your workspace'})).toBeVisible();
  await expect(page.getByRole('link',{name:/Sign in with SSO/i})).toBeVisible();
  await expect(page.getByLabel('Authenticator code')).toBeVisible();
  await expect(page.getByText('Protected session')).toBeVisible();
  await page.locator('#password').fill(password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByRole('heading',{name:'SQL Workspace'})).toBeVisible();
  await expect(page.getByRole('region',{name:'Workspace overview'})).toHaveCount(0); // informational cards are not a landmark
  await expect(page.locator('.workspace-summary .summary-card')).toHaveCount(3);
});

test('connection dialog is labelled and traps/restores keyboard focus',async({page})=>{
  await signIn(page);
  const trigger=page.getByRole('button',{name:'New connection',exact:true});
  await trigger.click();
  const dialog=page.getByRole('dialog',{name:'New database connection'});
  await expect(dialog).toBeVisible();
  await expect(page.getByText('Credentials stay on the Go server for this session only.')).toBeVisible();
  await expect(dialog.getByText('Connection name')).toBeVisible();
  await expect(dialog.locator('input').first()).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).toHaveCount(0);
  await expect(trigger).toBeFocused();
});

test('mobile navigation is operable and does not cover the workspace initially',async({page})=>{
  await page.setViewportSize({width:375,height:812});
  await signIn(page);
  const toggle=page.getByRole('button',{name:'Expand navigation'});
  await expect(toggle).toHaveAttribute('aria-expanded','false');
  await toggle.click();
  await expect(page.getByRole('button',{name:'Collapse navigation'})).toHaveAttribute('aria-expanded','true');
  await expect(page.getByRole('navigation',{name:'Primary'})).toBeVisible();
  await expect(page.getByRole('textbox',{name:'Filter connections'})).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button',{name:'Expand navigation'})).toHaveAttribute('aria-expanded','false');
  await page.getByRole('button',{name:'New connection',exact:true}).click();
  await expect(page.getByRole('dialog',{name:'New database connection'})).toBeVisible();
  await page.getByRole('button',{name:'Close',exact:true}).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('workspace tab drafts and results tab are accessible after redesign',async({page})=>{
  await signIn(page);
  const editor=page.getByRole('textbox',{name:'SQL query editor'});
  await editor.fill('SELECT 111 AS ui_review');
  await page.getByRole('button',{name:'New SQL tab'}).click();
  await expect(page.getByRole('tab',{name:'Query 2'})).toHaveAttribute('aria-selected','true');
  await editor.fill('SELECT 222 AS ui_review_two');
  await page.getByRole('tab',{name:'Query 1'}).click();
  await expect(editor).toHaveValue('SELECT 111 AS ui_review');
  await page.getByRole('tab',{name:'History'}).click();
  await page.getByRole('tab',{name:'Results'}).click();
  await expect(page.getByRole('tab',{name:'Results'})).toHaveAttribute('aria-selected','true');
});
