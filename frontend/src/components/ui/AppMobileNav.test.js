import { createSSRApp } from 'vue';
import { renderToString } from 'vue/server-renderer';
import { describe, expect, it } from 'vitest';
import AppMobileNav from './AppMobileNav.vue';

describe('mobile navigation ministry entry', () => {
  it('hides the ministry button when the current study group has no ministry groups', async () => {
    const html = await renderToString(createSSRApp(AppMobileNav, { tab: 'home', showGroups: false }));
    expect(html).not.toContain('>小组</span>');
    expect((html.match(/<button/g) || []).length).toBe(4);
  });

  it('shows the ministry button when ministry groups are available', async () => {
    const html = await renderToString(createSSRApp(AppMobileNav, { tab: 'groups', showGroups: true }));
    expect(html).toContain('>小组</span>');
    expect((html.match(/<button/g) || []).length).toBe(5);
  });

  it('shows the ministry button when the global setting is on, even with no groups', async () => {
    const html = await renderToString(createSSRApp(AppMobileNav, { tab: 'home', showGroups: false, entrySetting: true }));
    expect(html).toContain('>小组</span>');
  });

  it('hides the ministry button when the global setting is off, even with groups', async () => {
    const html = await renderToString(createSSRApp(AppMobileNav, { tab: 'home', showGroups: true, entrySetting: false }));
    expect(html).not.toContain('>小组</span>');
  });
});
