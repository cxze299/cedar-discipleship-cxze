import { createSSRApp, h } from 'vue';
import { renderToString } from 'vue/server-renderer';
import { afterEach, describe, expect, it, vi } from 'vitest';
import MobileCardCollection from './MobileCardCollection.vue';

const items = [{ id: 1, name: '甲' }, { id: 2, name: '乙' }];

async function renderCollection(mode) {
  return renderToString(createSSRApp({
    render: () => h(MobileCardCollection, {
      items,
      itemKey: (item) => item.id,
      mode,
      ariaLabel: '测试列表',
    }, {
      default: ({ item }) => h('div', { class: 'test-card' }, item.name),
    }),
  }));
}

describe('MobileCardCollection', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders the stacked wheel by default', async () => {
    vi.stubGlobal('window', {
      matchMedia: () => ({ matches: true }),
    });
    const html = await renderCollection(undefined);

    expect(html).toContain('stacked-wheel__stage');
    expect(html).not.toContain('mobile-card-collection__masonry');
  });

  it('preserves masonry as an explicit option', async () => {
    const html = await renderCollection('masonry');

    expect(html).toContain('mobile-card-collection__masonry');
    expect(html).not.toContain('stacked-wheel__stage');
    expect(html).toContain('甲');
    expect(html).toContain('乙');
  });
});
