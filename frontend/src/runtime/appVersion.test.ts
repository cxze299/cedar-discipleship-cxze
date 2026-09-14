import { afterEach, describe, expect, it, vi } from 'vitest';
import { refreshForNewVersion } from './appVersion';

describe('refreshForNewVersion', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('retries navigation when iOS restores the old bundle with the latest marker', async () => {
    const replace = vi.fn();
    vi.stubEnv('DEV', false);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ version: 'new-build' }),
    }));
    vi.stubGlobal('window', {
      location: {
        href: 'https://example.com/?_appv=new-build',
        replace,
      },
      history: {
        replaceState: vi.fn(),
      },
    });

    await expect(refreshForNewVersion()).resolves.toBe(true);
    expect(replace).toHaveBeenCalledOnce();
    expect(String(replace.mock.calls[0][0])).toContain('_appv_retry=1');
  });

  it('stops after two cache-busting reload attempts', async () => {
    const replace = vi.fn();
    vi.stubEnv('DEV', false);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ version: 'new-build' }),
    }));
    vi.stubGlobal('window', {
      location: {
        href: 'https://example.com/?_appv=new-build&_appv_retry=2',
        replace,
      },
      history: {
        replaceState: vi.fn(),
      },
    });

    await expect(refreshForNewVersion()).resolves.toBe(false);
    expect(replace).not.toHaveBeenCalled();
  });
});
