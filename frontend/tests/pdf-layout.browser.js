// From frontend/ with Vite on port 5174: playwright-cli run-code --filename=tests/pdf-layout.browser.js
async (page) => {
  const context = await page.context().browser().newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
  const test = await context.newPage();
  await test.goto('http://127.0.0.1:5174/tests/fixtures/pdf-layout.html');
  await test.locator('.pdf-viewer canvas').waitFor();

  const mobile = [];
  for (const width of [360, 390, 430]) {
    await test.setViewportSize({ width, height: 844 });
    const layout = await test.evaluate(() => {
      const toolbar = document.querySelector('.viewer-main-toolbar').getBoundingClientRect();
      const pdf = document.querySelector('.pdf-viewer').getBoundingClientRect();
      const canvas = document.querySelector('.pdf-viewer canvas').getBoundingClientRect();
      const sidebar = document.querySelector('.viewer-sidebar-desktop');
      return {
        viewportWidth: innerWidth,
        pageScrollWidth: document.documentElement.scrollWidth,
        toolbarBottom: toolbar.bottom,
        pdfTop: pdf.top,
        pdfWidth: pdf.width,
        canvasWidth: canvas.width,
        sidebarDisplay: getComputedStyle(sidebar).display,
      };
    });
    if (layout.sidebarDisplay !== 'none'
      || layout.pdfTop < layout.toolbarBottom - 2
      || layout.pdfWidth < layout.viewportWidth * 0.8
      || layout.canvasWidth < layout.viewportWidth * 0.6
      || layout.canvasWidth > layout.pdfWidth
      || layout.pageScrollWidth > layout.viewportWidth) {
      throw new Error(`Mobile PDF layout is cramped: ${JSON.stringify(layout)}`);
    }
    mobile.push(layout);
  }

  await test.setViewportSize({ width: 1200, height: 900 });
  const desktop = await test.evaluate(() => ({
    sidebarDisplay: getComputedStyle(document.querySelector('.viewer-sidebar-desktop')).display,
    pdfWidth: document.querySelector('.pdf-viewer').getBoundingClientRect().width,
  }));
  if (desktop.sidebarDisplay === 'none' || desktop.pdfWidth < 300) {
    throw new Error(`Desktop PDF layout changed: ${JSON.stringify(desktop)}`);
  }

  await page.evaluate((result) => { document.body.innerText = JSON.stringify(result); }, { mobile, desktop });
  await context.close();
}
