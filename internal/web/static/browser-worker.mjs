import { chromium } from "playwright-core";

const raw = process.argv[2];
const steps = JSON.parse(raw || "[]");
const profile = process.env.AGENT_V_BROWSER_PROFILE || `${process.env.HOME}/.config/agent-v/browser-profile`;
const executable = process.env.AGENT_V_CHROME || "/usr/bin/google-chrome";
let browser;
try {
  browser = await chromium.launchPersistentContext(profile, { headless: false, executablePath: executable });
  const page = browser.pages()[0] || await browser.newPage();
  let result = "Browser steps completed";
  for (const step of steps) {
    if (step.goto) await page.goto(step.goto, { waitUntil: "domcontentloaded", timeout: 30000 });
    else if (step.click) await page.locator(step.click).first().click({ timeout: 15000 });
    else if (step.fill) await page.locator(step.fill.selector).first().fill(String(step.fill.value ?? ""), { timeout: 15000 });
    else if (step.press) await page.keyboard.press(step.press);
    else if (step.wait) await page.waitForTimeout(Math.min(Number(step.wait), 10000));
    else if (step.expectText) await page.getByText(step.expectText, { exact: false }).first().waitFor({ state: "visible", timeout: 15000 });
    else if (step.title) result = await page.title();
    else throw new Error("Unsupported Playwright step");
  }
  console.log(JSON.stringify({ ok: true, result }));
} catch (error) {
  console.log(JSON.stringify({ ok: false, error: error?.message || String(error) }));
  process.exitCode = 1;
} finally {
  if (browser) await browser.close();
}
