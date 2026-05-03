import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') errors.push(msg.text());
  });
  page.on('pageerror', err => errors.push(err.message));
  
  await page.goto('http://localhost:3008', { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);
  
  const title = await page.title();
  console.log('Title:', title);
  
  const body = await page.evaluate(() => document.body.innerHTML.substring(0, 2000));
  console.log('Body:', body);
  
  console.log('Console errors:', errors);
  
  await browser.close();
})();
