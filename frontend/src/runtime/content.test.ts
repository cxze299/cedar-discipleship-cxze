import { describe, expect, it } from 'vitest';
import {
  applyPdfPageRangeToTitle,
  buildReaderPageURL,
  buildWeeklyVerseContentLink,
  classifyAttachment,
  deepMerge,
  enabledFlag,
  extractNumberedContentSection,
  extractWeeklyContentSection,
  extractPdfPageRange,
  inferAssetContentType,
  markdownToSafeHTML,
  normalizeSearchText,
  parsePdfPageRangeParts,
  parseReaderPageRequest,
  sameOriginAPIPath,
  shouldRenderWeeklyTask,
  videoMediaErrorMessage,
  weeklyTitleFromContent,
} from './content';

describe('content runtime helpers', () => {
  it('normalizes persisted boolean flags', () => {
    expect(enabledFlag('off')).toBe(false);
    expect(enabledFlag('yes')).toBe(true);
    expect(enabledFlag('', false)).toBe(false);
    expect(shouldRenderWeeklyTask(true, [{ id: 1 }])).toBe(true);
    expect(shouldRenderWeeklyTask(true, [])).toBe(false);
    expect(shouldRenderWeeklyTask(false, [{ id: 1 }])).toBe(false);
  });

  it('parses and normalizes PDF page ranges', () => {
    expect(extractPdfPageRange('阅读 12-18 页')).toBe('12-18');
    expect(extractPdfPageRange('圣经救赎史剧综览-2 196-198页')).toBe('196-198');
    expect(extractPdfPageRange('阅读 18 至 12 页')).toBe('18-18');
    expect(parsePdfPageRangeParts('第 9 页')).toEqual({ pageStart: '9', pageEnd: '9' });
    expect(applyPdfPageRangeToTitle('读物 3-4页', '8', '6')).toBe('读物 8-8页');
    expect(applyPdfPageRangeToTitle('读物 3-4页', '', '')).toBe('读物');
  });

  it('classifies attachments into previewable and download-only types', () => {
    expect(classifyAttachment({ filename: '主日信息.pdf' })).toEqual({ action: 'preview', type: 'pdf' });
    expect(inferAssetContentType({
      original_name: '圣经救赎史剧综览-2',
      mime_type: 'application/pdf',
    })).toBe('pdf');
    expect(inferAssetContentType({
      original_name: '圣经救赎史剧综览-2',
      category: 'book',
    })).toBe('pdf');
    expect(classifyAttachment({ filename: '录音.m4a' })).toEqual({ action: 'preview', type: 'audio' });
    expect(classifyAttachment({ mimeType: 'video/mp4', filename: '现场记录' })).toEqual({ action: 'preview', type: 'video' });
    expect(classifyAttachment({ filename: '服事安排.pptx' })).toEqual({ action: 'download', type: 'download' });
    expect(classifyAttachment({ filename: '成员清单.xlsx' })).toEqual({ action: 'download', type: 'download' });
    expect(classifyAttachment({ filename: '资料.unknown' })).toEqual({ action: 'download', type: 'download' });
  });

  it('describes media failures by browser error code', () => {
    expect(videoMediaErrorMessage(2)).toContain('网络');
    expect(videoMediaErrorMessage(3)).toContain('解码');
    expect(videoMediaErrorMessage(4)).toContain('不支持');
    expect(videoMediaErrorMessage(0)).toBe('视频加载失败，请重试');
  });

  it('keeps search matching and configuration merging deterministic', () => {
    expect(normalizeSearchText('《基督》：第一章')).toBe('基督第一章');
    expect(deepMerge(
      { daily: { enabled: true, path: '/default.md' }, items: [1] },
      { daily: { path: '/custom.md' }, items: [2, 3] },
    )).toEqual({
      daily: { enabled: true, path: '/custom.md' },
      items: [2, 3],
    });
  });

  it('builds weekly titles from enabled learning content', () => {
    expect(weeklyTitleFromContent({
      book_enabled: true,
      video_enabled: true,
      verse_enabled: true,
      readings: [{ title: '读物一' }, { title: '读物二' }],
      videos: [{ title: '视频一' }, { title: '视频二' }],
      verse_ref: '罗马书 8:1',
    })).toBe('读物一；读物二；视频一；罗马书 8:1');
    expect(weeklyTitleFromContent({
      title: '手动标题',
      readings: [{ title: '读物一' }],
    })).toBe('读物一');
  });

  it('builds an inline content link for weekly verse text', () => {
    expect(buildWeeklyVerseContentLink('罗马书 8:11-15', '罗马书 8:11 原文')).toEqual({
      label: '查看原文',
      title: '罗马书 8:11-15',
      type: 'markdown',
      content: '罗马书 8:11 原文',
    });
    expect(buildWeeklyVerseContentLink('罗马书 8:11-15', '  ')).toMatchObject({
      type: 'iframe',
      url: 'https://www.wordproject.org/bibles/gb/45/8.htm#11',
    });
    expect(buildWeeklyVerseContentLink('林前 13：4-8', '')).toMatchObject({
      url: 'https://www.wordproject.org/bibles/gb/46/13.htm#4',
    });
    expect(buildWeeklyVerseContentLink('未识别的经文', '')).toBeNull();
    expect(buildWeeklyVerseContentLink('罗马书 99:1', '')).toBeNull();
  });

  it('preserves the aggregate week title even with reading links disabled', () => {
    expect(weeklyTitleFromContent({
      weekly_checkin: true,
      title: '复习两个主题',
      book_enabled: false,
      verse_ref: '罗马书 8:1',
    })).toBe('复习两个主题');
  });

  it('accepts titled dates and respects explicit selection modes', () => {
    const text = '# 1\n无关数字篇章\n### 九月二十日 信心\n正文\n### 九月二十一日 次日\n后文';
    expect(extractNumberedContentSection(text, 0, '九月二十日')).toEqual([
      '### 九月二十日 信心', '正文',
    ]);
    expect(extractNumberedContentSection(text, 1, '九月二十日', 'date')).toEqual([
      '### 九月二十日 信心', '正文',
    ]);
    expect(extractNumberedContentSection(text, 2, '九月二十日', 'numbered')).toEqual([]);
    expect(extractNumberedContentSection('9月20日 标题\n正文\n九月二十一日\n后文', 0, '九月二十号')).toEqual([
      '9月20日 标题', '正文',
    ]);
    expect(extractNumberedContentSection(
      '九月廿九日\n前文。九月卅日「正文」\n十月一日\n后文',
      0,
      '九月三十日',
    )).toEqual(['九月卅日', '「正文」']);
    expect(extractNumberedContentSection('# 1\n正文', 1, '九月二十日', 'date')).toEqual([]);
  });

  it('matches legacy weekly themes including review titles and stops at the next chapter', () => {
    const text = '# 卷首语\n## 二、 基督是神的仆人--马可福音\n马可正文\n### 提纲\n内容\n## 五、 基督在身体里--使徒行传\n使徒正文\n## 六、 基督在福音里--罗马书\n后文';
    expect(extractWeeklyContentSection(text, '复习马可福音')).toEqual([
      '## 二、 基督是神的仆人--马可福音', '马可正文', '### 提纲', '内容',
    ]);
    expect(extractWeeklyContentSection(text, '《基督在身体里》使徒行传')).toEqual([
      '## 五、 基督在身体里--使徒行传', '使徒正文',
    ]);
    expect(extractWeeklyContentSection(text, '永活之泉')).toEqual([]);
    expect(extractWeeklyContentSection(text, '')).toEqual([]);
  });

  it('extracts numbered devotion content from numeric or Chinese date headings', () => {
    expect(extractNumberedContentSection(
      '# 186\n前一篇\n# 187\n目标正文\n# 188\n后一篇',
      187,
      '七月六号',
    )).toEqual(['# 187', '目标正文']);
    expect(extractNumberedContentSection(
      '卷首语\n\n七月五日\n前一篇\n\n七月六日\n目标正文\n第二段\n\n七月七日\n后一篇',
      187,
      '七月六号',
    )).toEqual(['七月六日', '目标正文', '第二段', '']);
  });

  it('renders markdown while escaping raw HTML and unsafe links', () => {
    const html = markdownToSafeHTML(
      '# 标题\n- **完成**\n<script>alert(1)</script>\n[安全](https://example.com)\n[读物](/api/assets/12/download)\n[无效路径](/files/unmanaged.pdf)\n[危险](javascript:alert(1))',
    );
    expect(html).toContain('<h2>标题</h2>');
    expect(html).toContain('<li><strong>完成</strong></li>');
    expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;');
    expect(html).toContain('href="https://example.com"');
    expect(html).toContain('href="/api/assets/12/download"');
    expect(html).not.toContain('href="/files/unmanaged.pdf"');
    expect(html).not.toContain('javascript:');
  });

  it('recognizes same-origin protected API URLs', () => {
    expect(sameOriginAPIPath('/api/assets/12/range?pages=10-11', 'http://localhost:5114')).toBe('/api/assets/12/range?pages=10-11');
    expect(sameOriginAPIPath('http://localhost:5114/api/assets/12/range?pages=10-11', 'http://localhost:5114')).toBe('/api/assets/12/range?pages=10-11');
    expect(sameOriginAPIPath('http://example.com/api/assets/12/range?pages=10-11', 'http://localhost:5114')).toBe('');
  });

  it('builds and parses protected PDF reader page URLs', () => {
    const url = buildReaderPageURL({
      sourceURL: '/api/assets/12/range?pages=10-11',
      title: '门训读物',
      pageRange: '10-11',
    }, 'http://localhost:5114');
    expect(url).toBe(
      'http://localhost:5114/?reader_source=%2Fapi%2Fassets%2F12%2Frange%3Fpages%3D10-11&reader_title=%E9%97%A8%E8%AE%AD%E8%AF%BB%E7%89%A9&reader_pages=10-11',
    );
    expect(parseReaderPageRequest(new URL(url).search)).toEqual({
      sourceURL: '/api/assets/12/range?pages=10-11',
      title: '门训读物',
      pageRange: '10-11',
    });
    expect(buildReaderPageURL({
      sourceURL: 'https://example.com/book.pdf',
      title: '外部文件',
      pageRange: '',
    }, 'http://localhost:5114')).toBe('');
    expect(buildReaderPageURL({
      sourceURL: '/api/assets/12/download',
      title: '整本文件',
      pageRange: '',
    }, 'http://localhost:5114')).toBe('');
    expect(parseReaderPageRequest('?reader_source=https://example.com/book.pdf')).toBeNull();
  });
});
