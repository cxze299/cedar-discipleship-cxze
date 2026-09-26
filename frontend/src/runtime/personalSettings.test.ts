import { describe, expect, it } from 'vitest';

import { normalizeMobileViewMode } from './personalSettings';

describe('normalizeMobileViewMode', () => {
  it.each([
    [undefined, 'masonry'],
    ['', 'masonry'],
    ['unknown', 'masonry'],
    ['masonry', 'masonry'],
    ['stacked', 'stacked'],
  ])('normalizes %j to %s', (value, expected) => {
    expect(normalizeMobileViewMode(value)).toBe(expected);
  });
});
