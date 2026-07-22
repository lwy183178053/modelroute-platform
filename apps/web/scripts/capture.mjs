import { mkdir, stat } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright-core';
import { startLocalPreview, stopLocalPreview } from './local-preview.mjs';

const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const baseUrl = process.env.UI_BASE_URL || 'http://127.0.0.1:4173';
const previewServer = await startLocalPreview();
const outputDir = fileURLToPath(new URL('../artifacts/', import.meta.url));
await mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({ executablePath, headless: true });

async function openNavigation(page, label) {
  const target = page.getByRole('button', { name: label, exact: true });
  if (!(await target.isVisible())) await page.getByRole('button', { name: '打开菜单' }).click();
  await target.click();
  await page.waitForTimeout(120);
}

async function verifyLayout(page, name) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
    textLength: document.body.innerText.trim().length,
  }));
  if (dimensions.scrollWidth > dimensions.clientWidth + 2) throw new Error(`${name} 根页面横向溢出 ${dimensions.scrollWidth - dimensions.clientWidth}px`);
  if (dimensions.textLength < 200) throw new Error(`${name} 页面内容疑似空白`);
}

async function captureSet(viewport, suffix) {
  const page = await browser.newPage({ viewport, deviceScaleFactor: 1 });
  const runtimeErrors = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  await page.goto(baseUrl, { waitUntil: 'networkidle' });

  const captures = [
    ['模型市场', 'market'],
    ['控制台', 'console'],
    ['商户中心', 'merchant'],
    ['管理后台', 'admin'],
  ];

  for (const [label, filename] of captures) {
    if (filename !== 'market') await openNavigation(page, label);
    if (filename === 'admin') await page.getByRole('button', { name: '支付与充值' }).click();
    await verifyLayout(page, `${filename}-${suffix}`);
    const path = `${outputDir}${filename}-${suffix}.png`;
    await page.screenshot({ path, fullPage: true });
    if ((await stat(path)).size < 20_000) throw new Error(`${filename}-${suffix} 截图文件过小`);
  }

  await page.getByRole('button', { name: '编辑配置' }).click();
  let path = `${outputDir}payment-config-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`payment-config-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  await openNavigation(page, '控制台');
  await page.getByRole('button', { name: '编辑路由策略' }).first().click();
  path = `${outputDir}route-modal-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`route-modal-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  if (runtimeErrors.length) throw new Error(`${suffix} 存在运行时错误：${runtimeErrors.join('; ')}`);
  await page.close();
}

try {
  await captureSet({ width: 1440, height: 1000 }, 'desktop');
  await captureSet({ width: 390, height: 844 }, 'mobile');
  console.log('截图完成：8 个业务视图、4 个关键弹窗，根页面无横向溢出。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
