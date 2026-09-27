import { createSSRApp } from 'vue';
import { renderToString } from 'vue/server-renderer';
import { createPinia } from 'pinia';
import { describe, expect, it } from 'vitest';
import Dashboard from './Dashboard.vue';
import { useDashboardStore } from '../stores/dashboard';

describe('dashboard statistics', () => {
  it('renders scripture and aggregate weekly counts alongside existing categories', async () => {
    const pinia = createPinia();
    useDashboardStore(pinia).setSnapshot({
      visible: true,
      ranking: [{
        user_id: 1, member_name: '成员甲', total: 6,
        counts: { daily_devotion: 1, daily_scripture: 2, weekly_checkin: 3 },
      }],
    });
    const context = {};
    await renderToString(createSSRApp(Dashboard).use(pinia), context);
    const html = context.teleports['#vue-dashboard'];
    expect(html).toContain('读经');
    expect(html).toContain('整周');
    expect(html).toContain('灵修');
    expect(html).toContain('成员甲');
  });
});
