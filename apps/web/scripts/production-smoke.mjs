import { mkdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright-core';

const baseUrl = process.env.PRODUCTION_BASE_URL || 'https://aggregate.lwylink.xyz';
const credentialPath = process.env.ADMIN_PASSWORD_FILE;
const outputDir = process.env.PRODUCTION_SCREENSHOT_DIR || path.join(process.env.TEMP, 'modelroute-production-ui');
const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
  || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';

if (!credentialPath) throw new Error('ADMIN_PASSWORD_FILE is required');

const credentials = {};
for (const line of (await readFile(credentialPath, 'utf8')).split(/\r?\n/)) {
  const separator = line.indexOf('=');
  if (separator > 0) credentials[line.slice(0, separator).trim()] = line.slice(separator + 1).trim();
}
if (!credentials.username || !credentials.password) throw new Error('credential file is incomplete');

await mkdir(outputDir, { recursive: true });

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function navigate(page, label, mobile) {
  if (mobile) await page.getByRole('button', { name: '打开菜单' }).click();
  await page.getByRole('navigation', { name: '主导航' })
    .getByRole('button', { name: label, exact: true })
    .click();
}

async function verifyViewport(browser, viewport, suffix) {
  const mobile = viewport.width < 700;
  const context = await browser.newContext({ viewport, locale: 'zh-CN' });
  const page = await context.newPage();
  const runtimeErrors = [];
  const serverErrors = [];

  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  page.on('console', (message) => {
    if (message.type() === 'error') runtimeErrors.push(message.text());
  });
  page.on('response', (response) => {
    if (response.status() >= 500) serverErrors.push(`${response.status()} ${response.url()}`);
  });

  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  const login = await page.evaluate(async (values) => {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    });
    return { status: response.status, body: await response.json() };
  }, { username: credentials.username, password: credentials.password });
  assert(
    login.status === 200 && login.body?.success === true,
    `${suffix} login failed: status=${login.status} code=${login.body?.error?.code || 'unknown'}`,
  );

  runtimeErrors.length = 0;
  serverErrors.length = 0;
  await page.reload({ waitUntil: 'networkidle' });
  await page.getByText('ModelRoute', { exact: true }).first().waitFor();
  await page.locator('nav[aria-label="主导航"]').waitFor({ state: 'attached' });
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1), `${suffix} root horizontal overflow`);

  await page.getByRole('button', { name: '公告', exact: true }).click();
  await page.getByRole('dialog', { name: '公告中心' }).waitFor();
  await page.locator('.announcement-list').waitFor();
  await page.getByRole('button', { name: '关闭公告' }).click();

  if (!mobile) {
    await page.locator('.profile-button').click();
    await page.getByRole('heading', { name: '账户概览' }).waitFor();
  }

  await navigate(page, '商户中心', mobile);
  await page.getByRole('heading', { name: '商户经营概览' }).waitFor();
  const merchantSidebar = page.getByRole('complementary', { name: '商户功能' });
  await merchantSidebar.waitFor();
  const merchantLabels = await merchantSidebar.getByRole('button').allTextContents();
  assert(
    merchantLabels.join('|') === '经营概览|渠道管理|模型报价|健康状态|结算账本|提现记录',
    `${suffix} merchant sidebar mismatch`,
  );
  await merchantSidebar.getByRole('button', { name: '模型报价', exact: true }).click();
  await page.getByRole('heading', { name: '模型报价' }).waitFor();
  let screenshot = path.join(outputDir, `merchant-${suffix}.png`);
  await page.screenshot({ path: screenshot, fullPage: true });
  assert((await stat(screenshot)).size > 15_000, `${suffix} merchant screenshot too small`);

  await navigate(page, '管理后台', mobile);
  await page.getByRole('heading', { name: '运营管理' }).waitFor();
  const adminSidebar = page.getByRole('complementary', { name: '管理员功能' });
  await adminSidebar.getByRole('button', { name: '模型与定价', exact: true }).click();
  await page.getByRole('heading', { name: '标准模型与基础价格' }).waitFor();
  await page.getByText(/^价格版本 v\d+$/).first().waitFor();
  await page.getByRole('button', { name: '发布价格', exact: true }).first().click();
  const dialog = page.getByRole('dialog', { name: '发布模型价格' });
  await dialog.waitFor();
  await dialog.getByText('每百万 Token 价格', { exact: true }).waitFor();
  await dialog.getByLabel('第 1 档输入价格').waitFor();
  assert(!(await dialog.textContent()).includes('倍率'), `${suffix} pricing dialog still exposes multiplier`);
  screenshot = path.join(outputDir, `admin-pricing-${suffix}.png`);
  await page.screenshot({ path: screenshot, fullPage: false });
  assert((await stat(screenshot)).size > 15_000, `${suffix} pricing screenshot too small`);

  assert(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1), `${suffix} final horizontal overflow`);
  assert(runtimeErrors.length === 0, `${suffix} runtime errors: ${runtimeErrors.join('; ')}`);
  assert(serverErrors.length === 0, `${suffix} server errors: ${serverErrors.join('; ')}`);
  console.log(`${suffix}=success merchant_sidebar=shared announcement=visible profile=${mobile ? 'responsive-hidden' : 'console-link'} pricing=absolute no_overflow=true`);
  await context.close();
}

const browser = await chromium.launch({ executablePath, headless: true });
try {
  await verifyViewport(browser, { width: 1440, height: 1000 }, 'desktop');
  await verifyViewport(browser, { width: 390, height: 844 }, 'mobile');
  console.log(`screenshots=${outputDir}`);
} finally {
  await browser.close();
}
