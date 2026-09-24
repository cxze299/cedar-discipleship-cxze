import { afterEach, describe, expect, it, vi } from 'vitest';
import { readComponentTheme } from './theme';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('readComponentTheme', () => {
  it('reads numeric component metrics from CSS tokens', () => {
    const values: Record<string, string> = {
      '--cd-primary': '#285c4d',
      '--cd-radius-base': '12px',
      '--cd-control-h-desktop': '44px',
    };
    vi.stubGlobal('document', { documentElement: {} });
    vi.stubGlobal('getComputedStyle', () => ({
      getPropertyValue: (name: string) => values[name] || '',
    }));

    const theme = readComponentTheme();

    expect(theme.token.colorPrimary).toBe('#285c4d');
    expect(theme.token.borderRadius).toBe(12);
    expect(theme.token.controlHeight).toBe(44);
  });

  it('omits invalid numeric tokens instead of passing NaN to Ant Design', () => {
    vi.stubGlobal('document', { documentElement: {} });
    vi.stubGlobal('getComputedStyle', () => ({
      getPropertyValue: () => '',
    }));

    const theme = readComponentTheme();

    expect(theme.token).not.toHaveProperty('borderRadius');
    expect(theme.token).not.toHaveProperty('controlHeight');
  });
});
