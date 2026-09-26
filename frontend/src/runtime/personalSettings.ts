export type MobileViewMode = 'masonry' | 'stacked';

export function normalizeMobileViewMode(value: unknown): MobileViewMode {
  return value === 'stacked' ? 'stacked' : 'masonry';
}
