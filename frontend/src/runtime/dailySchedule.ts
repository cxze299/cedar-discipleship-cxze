import { dayOffsetFrom } from './date';

export type DailyScheduleConfig = Record<string, unknown> & {
  schedule_history?: unknown;
};

function scheduleStart(config: DailyScheduleConfig, dateKeys: string[]): string {
  for (const key of dateKeys) {
    const value = String(config?.[key] || '').trim();
    if (/^\d{4}-\d{2}-\d{2}$/.test(value)) return value;
  }
  return '';
}

export function resolveEffectiveSchedule(
  config: DailyScheduleConfig,
  date: string,
  dateKeys: string[],
): DailyScheduleConfig {
  const history = Array.isArray(config?.schedule_history)
    ? config.schedule_history.filter((item): item is DailyScheduleConfig => (
      Boolean(item) && typeof item === 'object' && !Array.isArray(item)
    ))
    : [];
  const candidates = [...history, config]
    .map((item) => ({ item, start: scheduleStart(item, dateKeys) }))
    .filter((item) => item.start)
    .sort((left, right) => left.start.localeCompare(right.start));
  if (!candidates.length) return config || {};

  const selected = [...candidates].reverse().find((item) => item.start <= date) || candidates[0];
  return {
    ...config,
    ...selected.item,
    schedule_history: history,
  };
}

export function numberedSectionForDate(config: DailyScheduleConfig, date: string): number {
  const effective = resolveEffectiveSchedule(
    config,
    date,
    ['numbered_start_date', 'start_date'],
  );
  const startDate = String(effective.numbered_start_date || effective.start_date || date);
  const startSection = Math.max(1, Number(effective.numbered_start || effective.start_section || 1));
  return startSection + Math.max(0, dayOffsetFrom(startDate, date));
}

export type ScriptureChapter = {
  bookName: string;
  bookId: string;
  chapter: number;
};

export function scriptureChaptersForDate(
  config: DailyScheduleConfig,
  date: string,
): ScriptureChapter[] {
  const effective = resolveEffectiveSchedule(config, date, ['start_date']);
  if (String(effective.type || '') === 'checkin') return [];
  const startDate = String(effective.start_date || date);
  const chaptersPerDay = Math.max(1, Number(effective.chapters_per_day || 1));
  const books = normalizeScriptureBooks(effective);
  if (!books.length) return [];

  let offset = Math.max(0, dayOffsetFrom(startDate, date)) * chaptersPerDay;
  let firstChapter = Math.max(1, Number(effective.start_chapter || 1));
  for (let index = 0; index < books.length; index += 1) {
    const book = books[index];
    const startChapter = index === 0 ? firstChapter : 1;
    const available = Math.max(0, book.chapters - startChapter + 1);
    if (offset >= available) {
      offset -= available;
      continue;
    }
    const result: ScriptureChapter[] = [];
    let chapter = startChapter + offset;
    for (let currentIndex = index; currentIndex < books.length && result.length < chaptersPerDay; currentIndex += 1) {
      const current = books[currentIndex];
      const currentStart = currentIndex === index ? chapter : 1;
      for (let value = currentStart; value <= current.chapters && result.length < chaptersPerDay; value += 1) {
        result.push({ bookName: current.bookName, bookId: current.bookId, chapter: value });
      }
      chapter = 1;
    }
    return result;
  }
  return [];
}

function normalizeScriptureBooks(config: DailyScheduleConfig) {
  const source = Array.isArray(config.books) && config.books.length
    ? config.books
    : (Array.isArray(config.sequence) && config.sequence.length
      ? config.sequence
      : [{
        book: config.book,
        book_id: config.book_id,
        chapters: config.max_chapters,
      }]);
  return source
    .filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object' && !Array.isArray(item))
    .map((item) => ({
      bookName: String(item.book || config.book || '').trim(),
      bookId: String(item.book_id || config.book_id || '').trim(),
      chapters: Math.max(0, Number(item.chapters || config.max_chapters || 0)),
    }))
    .filter((item) => item.bookName && item.bookId && item.chapters > 0);
}
