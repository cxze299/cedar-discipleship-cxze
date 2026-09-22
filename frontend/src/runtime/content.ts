export type PlainRecord = Record<string, unknown>;

export function enabledFlag(value: unknown, fallback = true): boolean {
  if (value === undefined || value === null || value === '') return fallback;
  if (value === false || value === 0) return false;
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    if (['0', 'false', 'no', 'off'].includes(normalized)) return false;
    if (['1', 'true', 'yes', 'on'].includes(normalized)) return true;
  }
  return Boolean(value);
}

export function shouldRenderWeeklyTask(enabled: unknown, tasks: unknown): boolean {
  return enabledFlag(enabled) && Array.isArray(tasks) && tasks.length > 0;
}

export function sameOriginAPIPath(value: unknown, origin = ''): string {
  const source = String(value || '').trim();
  if (source.startsWith('/api/')) return source;
  if (!origin) return '';
  try {
    const parsed = new URL(source, origin);
    if (parsed.origin !== origin || !parsed.pathname.startsWith('/api/')) return '';
    return `${parsed.pathname}${parsed.search}`;
  } catch {
    return '';
  }
}

export type ReaderPageRequest = {
  sourceURL: string;
  title: string;
  pageRange: string;
};

export function buildReaderPageURL(input: ReaderPageRequest, origin = ''): string {
  const sourceURL = sameOriginAPIPath(input.sourceURL, origin);
  if (!/^\/api\/assets\/\d+\/range\?/.test(sourceURL)) return '';
  if (!new URLSearchParams(sourceURL.split('?')[1] || '').get('pages')) return '';
  const url = new URL('/', origin || 'http://localhost');
  url.searchParams.set('reader_source', sourceURL);
  url.searchParams.set('reader_title', String(input.title || 'PDF 资料').trim() || 'PDF 资料');
  if (input.pageRange) url.searchParams.set('reader_pages', input.pageRange);
  return origin ? url.toString() : `${url.pathname}${url.search}`;
}

export function parseReaderPageRequest(search: unknown): ReaderPageRequest | null {
  const params = new URLSearchParams(String(search || ''));
  const sourceURL = params.get('reader_source') || '';
  if (!/^\/api\/assets\/\d+\/range\?/.test(sourceURL)) return null;
  if (!new URLSearchParams(sourceURL.split('?')[1] || '').get('pages')) return null;
  return {
    sourceURL,
    title: (params.get('reader_title') || 'PDF 资料').trim() || 'PDF 资料',
    pageRange: params.get('reader_pages') || '',
  };
}

export function assetDownloadURLWithPageRange(value: unknown, pageRange: unknown, origin = ''): string {
  const originalURL = String(value || '').trim();
  const apiPath = sameOriginAPIPath(originalURL, origin);
  const sourceURL = apiPath || originalURL;
  const range = resolvePdfPageRange({ pageRange });
  const assetMatch = sourceURL.match(/^\/api\/assets\/(\d+)\/download$/);
  if (!assetMatch || !range) return sourceURL;
  return `/api/assets/${assetMatch[1]}/range?pages=${encodeURIComponent(range)}`;
}

export function normalizeContentViewerType(
  value: unknown,
  sourceURL: unknown = '',
  pageRange: unknown = '',
  origin = '',
): string {
  const type = String(value || '').trim().toLowerCase();
  if (['book', 'mentor', 'passage'].includes(type)) return 'pdf';

  const apiPath = sameOriginAPIPath(sourceURL, origin) || String(sourceURL || '');
  const hasAssetPageRange = Boolean(resolvePdfPageRange({ pageRange }))
    && /^\/api\/assets\/\d+\/(?:download|range)\b/.test(apiPath);
  if (hasAssetPageRange && ['', 'download', 'iframe'].includes(type)) return 'pdf';
  return type;
}

export type AttachmentPresentation = {
  action: 'preview' | 'download';
  type: 'pdf' | 'image' | 'video' | 'audio' | 'markdown' | 'download';
};

export function videoMediaErrorMessage(code: unknown): string {
  switch (Number(code || 0)) {
    case 2:
      return '视频网络请求失败，请检查网络后重试';
    case 3:
      return '视频解码或音频输出失败，请刷新页面或更换浏览器重试';
    case 4:
      return '当前浏览器不支持此视频格式';
    default:
      return '视频加载失败，请重试';
  }
}

export function classifyAttachment(input: { filename?: unknown; mimeType?: unknown }): AttachmentPresentation {
  const filename = String(input.filename || '').trim().toLowerCase();
  const mimeType = String(input.mimeType || '').trim().toLowerCase();

  if (mimeType.includes('pdf') || filename.endsWith('.pdf')) return { action: 'preview', type: 'pdf' };
  if (mimeType.startsWith('image/') || /\.(avif|gif|jpe?g|png|svg|webp)$/.test(filename)) return { action: 'preview', type: 'image' };
  if (mimeType.startsWith('video/') || /\.(m4v|mov|mp4|webm)$/.test(filename)) return { action: 'preview', type: 'video' };
  if (mimeType.startsWith('audio/') || /\.(aac|flac|m4a|ma4|mp3|ogg|opus|wav|weba)$/.test(filename)) return { action: 'preview', type: 'audio' };
  if (mimeType.includes('markdown') || mimeType.startsWith('text/') || /\.(markdown|md|txt)$/.test(filename)) {
    return { action: 'preview', type: 'markdown' };
  }
  return { action: 'download', type: 'download' };
}

export function inferAssetContentType(input: {
  type?: unknown;
  original_name?: unknown;
  title?: unknown;
  mime_type?: unknown;
  category?: unknown;
}, fallback = 'iframe'): string {
  const explicitType = String(input.type || '').trim().toLowerCase();
  if (['book', 'mentor', 'passage'].includes(explicitType)) return 'pdf';
  if (explicitType) return explicitType;
  const attachment = classifyAttachment({
    filename: input.original_name || input.title,
    mimeType: input.mime_type,
  });
  if (attachment.action === 'preview') return attachment.type;
  switch (String(input.category || '').trim().toLowerCase()) {
    case 'book':
    case 'mentor':
    case 'passage':
      return 'pdf';
    case 'markdown':
      return 'markdown';
    case 'outline':
      return 'image';
    case 'video':
      return 'video';
    default:
      return fallback;
  }
}

export function inferDailyDevotionContentType(
  config: { type?: unknown; path?: unknown } = {},
  asset?: {
    id?: unknown;
    type?: unknown;
    original_name?: unknown;
    title?: unknown;
    mime_type?: unknown;
    category?: unknown;
  },
): '' | 'markdown' | 'pdf' {
  if (asset) {
    const explicitType = String(asset.type || '').trim().toLowerCase();
    const assetType = inferAssetContentType({
      ...asset,
      type: explicitType === 'pdf' || explicitType === 'markdown' ? explicitType : '',
    }, '');
    if (assetType === 'pdf' || assetType === 'markdown') return assetType;
    if (!config.path && !config.type) return '';
  }

  const pathType = classifyAttachment({ filename: config.path });
  if (pathType.action === 'preview' && (pathType.type === 'pdf' || pathType.type === 'markdown')) {
    return pathType.type;
  }
  return String(config.type || '').trim().toLowerCase() === 'pdf' ? 'pdf' : 'markdown';
}

export function weeklyTitleFromContent(input: {
  title?: unknown;
  weekly_checkin?: unknown;
  book_enabled?: unknown;
  video_enabled?: unknown;
  verse_enabled?: unknown;
  readings?: Array<{ title?: unknown }>;
  videos?: Array<{ title?: unknown }>;
  verse_ref?: unknown;
}): string {
  if (enabledFlag(input.weekly_checkin, false)) return String(input.title || '').trim() || '周任务';
  const parts: string[] = [];
  if (enabledFlag(input.book_enabled)) {
    for (const reading of input.readings || []) {
      const title = String(reading?.title || '').trim();
      if (title) parts.push(title);
    }
  }
  if (enabledFlag(input.video_enabled)) {
    const videoTitle = (input.videos || [])
      .map((video) => String(video?.title || '').trim())
      .find(Boolean);
    if (videoTitle) parts.push(videoTitle);
  }
  if (enabledFlag(input.verse_enabled)) {
    const verseRef = String(input.verse_ref || '').trim();
    if (verseRef) parts.push(verseRef);
  }
  return parts.join('；');
}

export function buildWeeklyVerseContentLink(verseRef: unknown, reciteText: unknown) {
  const content = String(reciteText || '').trim();
  const title = String(verseRef || '').trim() || '本周背经';
  if (content) {
    return { label: '查看原文', title, type: 'markdown' as const, content };
  }
  const target = bibleReferenceTarget(verseRef);
  if (!target) return null;
  return { label: '查看原文', title, type: 'iframe' as const, url: target };
}

export function extractNumberedContentSection(
  text: unknown,
  number: unknown,
  heading: unknown = '',
  mode: unknown = '',
): string[] {
  const lines = contentSectionLines(text);
  const sectionNumber = Number(number);
  const selectionMode = String(mode || '').trim().toLowerCase();
  if (selectionMode !== 'date' && Number.isFinite(sectionNumber) && sectionNumber > 0) {
    const startPattern = new RegExp(`^#{1,6}\\s*${sectionNumber}\\s*$`);
    const stopPattern = /^#{1,6}\s*\d+\s*$/;
    const numbered = extractSection(lines, (line) => startPattern.test(line), (line) => stopPattern.test(line));
    if (numbered.length) return numbered;
  }

  if (selectionMode === 'numbered') return [];
  const targetHeading = dateHeadingKey(heading);
  if (!targetHeading) return [];
  return extractSection(
    lines,
    (line) => dateHeadingKey(line) === targetHeading,
    (line) => Boolean(dateHeadingKey(line)),
  );
}

export function extractWeeklyContentSection(text: unknown, title: unknown): string[] {
  const target = normalizeSearchText(title);
  if (!target) return [];
  const lines = contentSectionLines(text);
  return extractSection(
    lines,
    (line) => {
      if (!/^##\s+/.test(line)) return false;
      const heading = normalizeSearchText(line.replace(/^##\s+/, ''));
      return Boolean(heading && (heading.includes(target) || target.includes(heading)
        || [...bibleBookReferences].some(([name]) => target.includes(normalizeSearchText(name))
          && heading.includes(normalizeSearchText(name)))));
    },
    (line) => /^##\s+/.test(line),
  );
}

function extractSection(
  lines: string[],
  startsSection: (line: string) => boolean,
  startsNextSection: (line: string) => boolean,
): string[] {
  let capturing = false;
  const content: string[] = [];
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!capturing) {
      if (startsSection(line)) {
        capturing = true;
        content.push(rawLine);
      }
      continue;
    }
    if (startsNextSection(line) && !startsSection(line)) break;
    content.push(rawLine);
  }
  return content;
}

function normalizeSectionHeading(value: unknown): string {
  return String(value || '')
    .trim()
    .replace(/^\uFEFF/, '')
    .replace(/^#{1,6}\s*/, '')
    .replace(/号$/, '日');
}

function dateHeadingKey(value: unknown): string {
  const heading = normalizeSectionHeading(value);
  const match = heading.match(/^([一二三四五六七八九十廿卅\d]+)月([一二三四五六七八九十廿卅\d]+)[日号]/);
  if (!match) return '';
  const month = parseChineseNumber(match[1]);
  const day = parseChineseNumber(match[2]);
  if (month < 1 || month > 12 || day < 1 || day > 31) return '';
  return `${month}-${day}`;
}

function parseChineseNumber(value: string): number {
  if (/^\d+$/.test(value)) return Number(value);
  if (value.startsWith('廿')) return 20 + parseChineseNumber(value.slice(1));
  if (value.startsWith('卅')) return 30 + parseChineseNumber(value.slice(1));
  const digits: Record<string, number> = {
    零: 0, 一: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9,
  };
  if (value === '十') return 10;
  const [tens, ones] = value.split('十');
  if (value.includes('十')) return (tens ? digits[tens] : 1) * 10 + (ones ? digits[ones] : 0);
  return digits[value] || 0;
}

function contentSectionLines(text: unknown): string[] {
  const date = '[一二三四五六七八九十廿卅\\d]+月[一二三四五六七八九十廿卅\\d]+[日号]';
  return String(text || '')
    .replace(/\r/g, '')
    .replace(new RegExp(`([。！？.!?])(${date})(?=[「“])`, 'g'), '$1\n$2\n')
    .split('\n');
}

const bibleBookReferences: Array<[string, string, number, string[]]> = [
  ['创世记', '1', 50, ['创']], ['出埃及记', '2', 40, ['出']], ['利未记', '3', 27, ['利']],
  ['民数记', '4', 36, ['民']], ['申命记', '5', 34, ['申']], ['约书亚记', '6', 24, ['书']],
  ['士师记', '7', 21, ['士']], ['路得记', '8', 4, ['得']], ['撒母耳记上', '9', 31, ['撒上']],
  ['撒母耳记下', '10', 24, ['撒下']], ['列王纪上', '11', 22, ['王上']], ['列王纪下', '12', 25, ['王下']],
  ['历代志上', '13', 29, ['代上']], ['历代志下', '14', 36, ['代下']], ['以斯拉记', '15', 10, ['拉']],
  ['尼希米记', '16', 13, ['尼']], ['以斯帖记', '17', 10, ['斯']], ['约伯记', '18', 42, ['伯']],
  ['诗篇', '19', 150, ['诗']], ['箴言', '20', 31, ['箴']], ['传道书', '21', 12, ['传']],
  ['雅歌', '22', 8, ['歌']], ['以赛亚书', '23', 66, ['赛']], ['耶利米书', '24', 52, ['耶']],
  ['耶利米哀歌', '25', 5, ['哀']], ['以西结书', '26', 48, ['结']], ['但以理书', '27', 12, ['但']],
  ['何西阿书', '28', 14, ['何']], ['约珥书', '29', 3, ['珥']], ['阿摩司书', '30', 9, ['摩']],
  ['俄巴底亚书', '31', 1, ['俄']], ['约拿书', '32', 4, ['拿']], ['弥迦书', '33', 7, ['弥']],
  ['那鸿书', '34', 3, ['鸿']], ['哈巴谷书', '35', 3, ['哈']], ['西番雅书', '36', 3, ['番']],
  ['哈该书', '37', 2, ['该']], ['撒迦利亚书', '38', 14, ['亚']], ['玛拉基书', '39', 4, ['玛']],
  ['马太福音', '40', 28, ['太']], ['马可福音', '41', 16, ['可']], ['路加福音', '42', 24, ['路']],
  ['约翰福音', '43', 21, ['约']], ['使徒行传', '44', 28, ['徒']], ['罗马书', '45', 16, ['罗']],
  ['哥林多前书', '46', 16, ['林前']], ['哥林多后书', '47', 13, ['林后']], ['加拉太书', '48', 6, ['加']],
  ['以弗所书', '49', 6, ['弗']], ['腓立比书', '50', 4, ['腓']], ['歌罗西书', '51', 4, ['西']],
  ['帖撒罗尼迦前书', '52', 5, ['帖前']], ['帖撒罗尼迦后书', '53', 3, ['帖后']],
  ['提摩太前书', '54', 6, ['提前']], ['提摩太后书', '55', 4, ['提后']], ['提多书', '56', 3, ['多']],
  ['腓利门书', '57', 1, ['门']], ['希伯来书', '58', 13, ['来']], ['雅各书', '59', 5, ['雅']],
  ['彼得前书', '60', 5, ['彼前']], ['彼得后书', '61', 3, ['彼后']], ['约翰一书', '62', 5, ['约壹', '约一']],
  ['约翰二书', '63', 1, ['约贰', '约二']], ['约翰三书', '64', 1, ['约叁', '约三']],
  ['犹大书', '65', 1, ['犹']], ['启示录', '66', 22, ['启']],
];

function bibleReferenceTarget(value: unknown): string {
  const source = String(value || '').trim().replaceAll('：', ':').replace(/\s+/g, '');
  for (const [name, id, chapters, aliases] of bibleBookReferences) {
    for (const label of [name, ...aliases].sort((a, b) => b.length - a.length)) {
      if (!source.startsWith(label)) continue;
      const match = source.slice(label.length).match(/^(\d{1,3}):(\d{1,3})/);
      if (!match) continue;
      const chapter = Number(match[1]);
      const verse = Number(match[2]);
      if (chapter < 1 || chapter > chapters || verse < 1) return '';
      return `https://www.wordproject.org/bibles/gb/${id}/${chapter}.htm#${verse}`;
    }
  }
  return '';
}

export function normalizeSearchText(value: unknown): string {
  return String(value || '')
    .replace(/[《》【】（）()：:·,\-—–_]/g, ' ')
    .replace(/\s+/g, '')
    .toLowerCase();
}

export function extractPdfPageRange(value: unknown): string {
  const match = String(value || '').match(/(\d{1,4})\s*(?:[-~—–至到]\s*(\d{1,4}))?\s*页/);
  if (!match) return '';
  const start = Math.max(1, Number(match[1] || 1));
  const end = Math.max(start, Number(match[2] || match[1] || start));
  return `${start}-${end}`;
}

function normalizePdfPageRange(value: unknown): string {
  const raw = String(value ?? '').trim();
  const match = raw.match(/^(\d{1,4})(?:\s*[-~—–至到]\s*(\d{1,4}))?$/);
  if (!match) return extractPdfPageRange(raw);
  const start = Math.max(1, Number(match[1] || 1));
  const end = Math.max(start, Number(match[2] || match[1] || start));
  return `${start}-${end}`;
}

export function extractPdfPageRangeFromMetadata(value: unknown): string {
  let metadata: PlainRecord | null = null;
  if (isPlainObject(value)) {
    metadata = value;
  } else {
    const raw = String(value || '').trim();
    if (!raw.startsWith('{')) return '';
    try {
      const parsed = JSON.parse(raw);
      metadata = isPlainObject(parsed) ? parsed : null;
    } catch {
      return '';
    }
  }
  if (!metadata) return '';

  for (const key of ['source_title', 'sourceTitle', 'title']) {
    const pageRange = extractPdfPageRange(metadata[key]);
    if (pageRange) return pageRange;
  }
  return composePdfPageRange(
    metadata.page_start ?? metadata.pageStart,
    metadata.page_end ?? metadata.pageEnd,
  );
}

export function resolvePdfPageRange(...sources: unknown[]): string {
  for (const source of sources) {
    if (isPlainObject(source)) {
      for (const key of ['pageRange', 'page_range', 'pages']) {
        const pageRange = normalizePdfPageRange(source[key]);
        if (pageRange) return pageRange;
      }
      const fieldRange = composePdfPageRange(
        source.page_start ?? source.pageStart,
        source.page_end ?? source.pageEnd,
      );
      if (fieldRange) return fieldRange;
      for (const key of ['source_title', 'sourceTitle', 'title', 'label', 'detail', 'part']) {
        const pageRange = extractPdfPageRange(source[key]);
        if (pageRange) return pageRange;
      }
      const metadataRange = extractPdfPageRangeFromMetadata(source.content ?? source.metadata);
      if (metadataRange) return metadataRange;
      continue;
    }

    const pageRange = extractPdfPageRange(source);
    if (pageRange) return pageRange;
    const metadataRange = extractPdfPageRangeFromMetadata(source);
    if (metadataRange) return metadataRange;
  }
  return '';
}

export function parsePdfPageRangeParts(value: unknown): { pageStart: string; pageEnd: string } {
  const pageRange = extractPdfPageRange(value);
  if (!pageRange) return { pageStart: '', pageEnd: '' };
  const [pageStart, pageEnd] = pageRange.split('-');
  return {
    pageStart: pageStart || '',
    pageEnd: pageEnd || pageStart || '',
  };
}

export function normalizePageField(value: unknown): string {
  const parsed = Number(String(value ?? '').trim());
  if (!Number.isFinite(parsed) || parsed < 1) return '';
  return String(Math.floor(parsed));
}

export function composePdfPageRange(startValue: unknown, endValue: unknown): string {
  const start = Number(normalizePageField(startValue));
  if (!start) return '';
  const end = Math.max(start, Number(normalizePageField(endValue) || start));
  return `${start}-${end}`;
}

export function applyPdfPageRangeToTitle(
  title: unknown,
  startValue: unknown,
  endValue: unknown,
): string {
  const source = String(title || '').trim();
  const pageRange = composePdfPageRange(startValue, endValue);
  const pageRegex = /(\d{1,4})\s*(?:[-~—–至到]\s*(\d{1,4}))?\s*页/;
  const stripRegex = /\s*(\d{1,4})\s*(?:[-~—–至到]\s*(\d{1,4}))?\s*页/g;
  if (!pageRange) {
    return source.replace(stripRegex, ' ').replace(/\s{2,}/g, ' ').trim();
  }
  const nextRange = `${pageRange}页`;
  if (pageRegex.test(source)) {
    return source.replace(pageRegex, nextRange).replace(/\s{2,}/g, ' ').trim();
  }
  return source ? `${source} ${nextRange}` : nextRange;
}

export function isPlainObject(value: unknown): value is PlainRecord {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

export function deepMerge<T>(base: T, override: unknown): T {
  if (Array.isArray(base)) {
    return (Array.isArray(override) ? override.slice() : base.slice()) as T;
  }
  if (!isPlainObject(base)) {
    return (isPlainObject(override) ? { ...override } : (override ?? base)) as T;
  }

  const result: PlainRecord = { ...base };
  const entries = isPlainObject(override) ? Object.entries(override) : [];
  for (const [key, value] of entries) {
    const baseValue = base[key];
    if (Array.isArray(value)) result[key] = value.slice();
    else if (isPlainObject(value) && isPlainObject(baseValue)) result[key] = deepMerge(baseValue, value);
    else if (isPlainObject(value)) result[key] = deepMerge({}, value);
    else result[key] = value;
  }
  return result as T;
}

export function markdownToSafeHTML(value: unknown): string {
  const escaped = escapeHTML(String(value || '').replace(/\r/g, ''));
  const lines = escaped.split('\n');
  const output: string[] = [];
  let listOpen = false;

  const closeList = () => {
    if (!listOpen) return;
    output.push('</ul>');
    listOpen = false;
  };

  for (const rawLine of lines) {
    const line = rawLine.trim();
    const listMatch = line.match(/^[-*]\s+(.+)$/);
    if (listMatch) {
      if (!listOpen) {
        output.push('<ul>');
        listOpen = true;
      }
      output.push(`<li>${inlineMarkdown(listMatch[1])}</li>`);
      continue;
    }
    closeList();
    if (!line) {
      output.push('');
    } else if (line.startsWith('### ')) {
      output.push(`<h4>${inlineMarkdown(line.slice(4))}</h4>`);
    } else if (line.startsWith('## ')) {
      output.push(`<h3>${inlineMarkdown(line.slice(3))}</h3>`);
    } else if (line.startsWith('# ')) {
      output.push(`<h2>${inlineMarkdown(line.slice(2))}</h2>`);
    } else if (line.startsWith('&gt; ')) {
      output.push(`<blockquote>${inlineMarkdown(line.slice(5))}</blockquote>`);
    } else {
      output.push(`<p>${inlineMarkdown(line)}</p>`);
    }
  }
  closeList();
  return output.join('');
}

function inlineMarkdown(value: string): string {
  return value
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`(.+?)`/g, '<code>$1</code>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, label: string, href: string) => {
      const decoded = href.replaceAll('&amp;', '&').trim();
      if (!/^(https?:\/\/|\/api\/assets\/\d+\/download$)/i.test(decoded)) {
        return label;
      }
      return `<a href="${escapeAttribute(decoded)}" target="_blank" rel="noopener noreferrer">${label}</a>`;
    });
}

function escapeHTML(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function escapeAttribute(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}
