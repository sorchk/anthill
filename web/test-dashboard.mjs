import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') errors.push(msg.text());
  });
  
  await page.goto('http://localhost:3008', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  
  // Fill login form
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  
  await page.waitForTimeout(3000);
  
  const body = await page.evaluate(() => document.body.innerHTML.substring(0, 3000));
  console.log('After login body:', body);
  
  console.log('Console errors:', errors);
  
  await browser.close();
})();
