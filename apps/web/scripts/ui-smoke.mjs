import { chromium } from 'playwright-core';
import { startLocalPreview, stopLocalPreview } from './local-preview.mjs';

const executablePath =
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ||
  'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const baseUrl = process.env.UI_BASE_URL || 'http://127.0.0.1:4173';
const previewServer = await startLocalPreview();

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

const browser = await chromium.launch({ executablePath, headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });

try {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.getByRole('heading', { name: '模型服务市场' }).waitFor();

  const views = [
    ['控制台', '账户概览'],
    ['商户中心', '商户经营概览'],
    ['管理后台', '运营管理'],
    ['模型市场', '模型服务市场'],
  ];

  for (const [navigation, heading] of views) {
    await page.getByRole('button', { name: navigation, exact: true }).click();
    await page.getByRole('heading', { name: heading }).waitFor();
  }

  await page.getByRole('button', { name: '控制台', exact: true }).click();
  await page.getByRole('button', { name: '编辑路由策略' }).first().click();
  await page.getByRole('dialog', { name: '编辑路由策略' }).waitFor();
  await page.getByLabel('固定商户').check();
  assert(await page.getByText('固定商户范围').isVisible(), '固定路由商户选择未显示');
  await page.getByRole('button', { name: '保存策略' }).click();

  await page.getByRole('button', { name: '商户中心', exact: true }).click();
  await page.getByRole('button', { name: '调整报价' }).first().click();
  await page.getByRole('dialog', { name: '调整模型报价' }).waitFor();
  await page.getByRole('button', { name: '提交调价' }).click();

  await page.getByRole('button', { name: '管理后台', exact: true }).click();
  await page.getByRole('button', { name: '支付与充值' }).click();
  await page.getByRole('heading', { name: '易支付设置' }).waitFor();
  assert(
    await page.getByText('https://zpayz.cn/').isVisible(),
    '易支付网关未显示',
  );
  const bodyText = await page.locator('body').innerText();
  assert(!/KEY[^\n]{0,24}[A-Za-z0-9]{24,}/.test(bodyText), '页面疑似显示了完整支付密钥');
  await page.getByRole('button', { name: '测试连接' }).click();
  await page.getByText('网关连接正常').waitFor();

  console.log('UI 冒烟测试通过：四个视图、路由策略、报价和支付设置交互正常。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
