
<script setup>
import { computed, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useAppStateStore } from '../stores/appState';
import { useDownloadManagerStore } from '../stores/downloadManager';
import {
  RESOURCE_UPLOAD_CATEGORIES,
  isWeeklyMediaResource,
  normalizeResourceCategory,
} from '../runtime/resources';
import MinistryCatalogAdmin from './MinistryCatalogAdmin.vue';
import BotManagementAdmin from './BotManagementAdmin.vue';
import ResourceGovernance from './ResourceGovernance.vue';
import DateField from './ui/DateField.vue';
import {
  api,
  addWeekBinding,
  applyBindingSelection,
  applyOutlineSelection,
  deleteWeekDraft,
  downloadAdminExport,
  enabledFlag,
  importLocalBackupJSON,
  importStudyWeeksExcel,
  librarySelectionValue,
  loadAdminData,
  removeWeekBinding,
  restoreWeekDraftDefaults,
  saveLearningConfig,
  saveWeekDraft,
  selectWeekDraft,
  setAdminSection,
  toast as showToast,
  updateLearningValue,
  updateWeekBinding,
  updateWeekDraftField,
  uploadLibraryFile,
  weekBindingSelectionValue,
  reloadApp,
} from '../legacy-app';

const app = useAppStateStore();
const {
  adminSection,
  canEditLearning,
  canEditStudyWeeks,
  learningConfig,
  weekDraft,
  weeks,
  resourceLibrary,
  user,
} = storeToRefs(app);

const uploadCategory = ref('markdown');
const uploadInput = ref(null);
const studyWeeksImportInput = ref(null);
const localBackupImportInput = ref(null);
const notificationSaving = ref(false);

function navigateTabs(event) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
  const tabs = Array.from(event.currentTarget.querySelectorAll('[role="tab"]'));
  const index = tabs.indexOf(event.target);
  if (index < 0) return;
  event.preventDefault();
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1
    : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
  tabs[next].focus();
  tabs[next].click();
}

const canManageMinistryCatalog = computed(() => Boolean(user.value?.is_super_admin || user.value?.roles?.includes('group_admin')));
const settings = computed(() => learningConfig.value || {});
const daily = computed(() => settings.value.task_sections?.daily || {});
const devotion = computed(() => daily.value.devotion || {});
const scripture = computed(() => daily.value.scripture || {});
const checkinNotifications = computed(() => settings.value.checkin_notifications || {});

const bibleBooks = [
  ['创世记', 50], ['出埃及记', 40], ['利未记', 27], ['民数记', 36], ['申命记', 34],
  ['约书亚记', 24], ['士师记', 21], ['路得记', 4], ['撒母耳记上', 31], ['撒母耳记下', 24],
  ['列王纪上', 22], ['列王纪下', 25], ['历代志上', 29], ['历代志下', 36], ['以斯拉记', 10],
  ['尼希米记', 13], ['以斯帖记', 10], ['约伯记', 42], ['诗篇', 150], ['箴言', 31],
  ['传道书', 12], ['雅歌', 8], ['以赛亚书', 66], ['耶利米书', 52], ['耶利米哀歌', 5],
  ['以西结书', 48], ['但以理书', 12], ['何西阿书', 14], ['约珥书', 3], ['阿摩司书', 9],
  ['俄巴底亚书', 1], ['约拿书', 4], ['弥迦书', 7], ['那鸿书', 3], ['哈巴谷书', 3],
  ['西番雅书', 3], ['哈该书', 2], ['撒迦利亚书', 14], ['玛拉基书', 4], ['马太福音', 28],
  ['马可福音', 16], ['路加福音', 24], ['约翰福音', 21], ['使徒行传', 28], ['罗马书', 16],
  ['哥林多前书', 16], ['哥林多后书', 13], ['加拉太书', 6], ['以弗所书', 6], ['腓立比书', 4],
  ['歌罗西书', 4], ['帖撒罗尼迦前书', 5], ['帖撒罗尼迦后书', 3], ['提摩太前书', 6], ['提摩太后书', 4],
  ['提多书', 3], ['腓利门书', 1], ['希伯来书', 13], ['雅各书', 5], ['彼得前书', 5],
  ['彼得后书', 3], ['约翰一书', 5], ['约翰二书', 1], ['约翰三书', 1], ['犹大书', 1], ['启示录', 22],
].map(([book, chapters], index) => ({ book, book_id: String(index + 1), chapters }));

const scriptureBookOptions = computed(() => bibleBooks);
const libraryItems = computed(() => resourceLibrary.value.flatMap((section) => section.items || []));

const markdownFileOptions = computed(() => {
  const seen = new Set();
  return libraryItems.value.filter((item) => {
    if (item.type !== 'markdown' || !item.url || seen.has(item.url)) return false;
    seen.add(item.url);
    return true;
  });
});

const readingOptions = computed(() => libraryItems.value.filter((item) => (
  ['book', 'passage', 'markdown'].includes(normalizeResourceCategory(item.category))
)));
const videoOptions = computed(() => libraryItems.value.filter(isWeeklyMediaResource));
const outlineOptions = computed(() => libraryItems.value.filter((item) => (
  item.type === 'image' || item.type === 'outline' || item.category === 'outline'
)));

function updateLearning(path, value) {
  updateLearningValue(path, value);
}

async function setCheckinNotification(key, enabled) {
  const previous = checkinNotifications.value[key] !== false;
  notificationSaving.value = true;
  updateLearning(['checkin_notifications', key], enabled);
  const saved = await saveLearningConfig('通知设置已保存');
  if (!saved) updateLearning(['checkin_notifications', key], previous);
  notificationSaving.value = false;
}

function updateScriptureBook(bookID) {
  const selected = scriptureBookOptions.value.find((item) => String(item.book_id) === String(bookID));
  if (!selected) return;
  const startIndex = bibleBooks.findIndex((item) => item.book_id === selected.book_id);
  updateLearning(['task_sections', 'daily', 'scripture'], {
    ...scripture.value,
    book: selected.book || scripture.value.book || '',
    book_id: selected.book_id || scripture.value.book_id || '',
    max_chapters: Number(selected.chapters || scripture.value.max_chapters || 1),
    sequence: bibleBooks.slice(startIndex),
  });
}

function optionText(item) {
  return item.title || item.original_name || '未命名资源';
}

function singleLineText(value) {
  return String(value || '').replace(/\s+/g, ' ').trim();
}

function weekOptionText(week) {
  const start = singleLineText(week?.start);
  const end = singleLineText(week?.end);
  const range = start && end ? `${start} - ${end}` : start || end || '未设置时间';
  const title = singleLineText(week?.title) || '未命名周任务';
  return `${range}｜${title}`;
}

function fileOptionText(item) {
  return item.title || item.original_name || item.url || '未命名文件';
}

function markdownOptionsWithCurrent(currentValue) {
  const current = String(currentValue || '').trim();
  if (!current || markdownFileOptions.value.some((item) => item.url === current)) {
    return markdownFileOptions.value;
  }
  return [{ title: `${current}（当前配置）`, url: current, type: 'markdown' }, ...markdownFileOptions.value];
}

async function uploadSelectedFile() {
  await uploadLibraryFile(uploadInput.value, uploadCategory.value);
}

async function runAdminExport(path, fallbackName, successMessage) {
  try {
    await downloadAdminExport(path, fallbackName, successMessage);
  } catch (error) {
    showToast(error.message);
  }
}

async function runStudyWeeksImport() {
  try {
    await importStudyWeeksExcel(studyWeeksImportInput.value);
  } catch (error) {
    showToast(error.message);
  }
}

async function runLocalBackupImport() {
  try {
    await importLocalBackupJSON(localBackupImportInput.value);
  } catch (error) {
    showToast(error.message);
  }
}
</script>

<template>
  <div class="admin-wrapper">
    <div class="pagehead spread admin-pagehead">
      <div>
        <h1>管理工作台</h1>
        <p class="muted">安排学习内容，管理小组资料</p>
      </div>
    </div>

    <!-- Secondary Nav Toolbar -->
    <div class="toolbar admin-tabs" role="tablist" aria-label="管理工作台功能" @keydown="navigateTabs">
      <button
        :class="adminSection === 'learning' ? 'primary' : 'quiet'"
        type="button"
        role="tab"
        :aria-selected="adminSection === 'learning'"
        :tabindex="adminSection === 'learning' ? 0 : -1"
        @click="setAdminSection('learning')"
      >
        学习配置
      </button>
      <button
        :class="adminSection === 'library' ? 'primary' : 'quiet'"
        type="button"
        role="tab"
        :aria-selected="adminSection === 'library'"
        :tabindex="adminSection === 'library' ? 0 : -1"
        @click="setAdminSection('library')"
      >
        资源管理
      </button>
      <button
        :class="adminSection === 'data' ? 'primary' : 'quiet'"
        type="button"
        role="tab"
        :aria-selected="adminSection === 'data'"
        :tabindex="adminSection === 'data' ? 0 : -1"
        @click="setAdminSection('data')"
      >
        数据管理
      </button>
      <button
        v-if="canManageMinistryCatalog"
        :class="adminSection === 'ministry' ? 'primary' : 'quiet'"
        type="button"
        role="tab"
        :aria-selected="adminSection === 'ministry'"
        :tabindex="adminSection === 'ministry' ? 0 : -1"
        @click="setAdminSection('ministry')"
      >
        专项小组
      </button>
      <button
        v-if="user?.is_super_admin"
        :class="adminSection === 'bot' ? 'primary' : 'quiet'"
        type="button"
        role="tab"
        :aria-selected="adminSection === 'bot'"
        :tabindex="adminSection === 'bot' ? 0 : -1"
        @click="setAdminSection('bot')"
      >
        机器人管理
      </button>
    </div>

    <MinistryCatalogAdmin v-if="adminSection === 'ministry' && canManageMinistryCatalog" />
    <BotManagementAdmin v-else-if="adminSection === 'bot' && user?.is_super_admin" />

    <section v-else-if="adminSection === 'learning'">
              <div class="grid admin-learning-stack">
                <div class="card">
                  <h2>打卡通知</h2>
                  <div class="admin-checkbox-row notification-toggle-row">
                    <label class="admin-toggle">
                      <input
                        type="checkbox"
                        :checked="checkinNotifications.daily_enabled !== false"
                        :disabled="!canEditLearning || notificationSaving"
                        @change="setCheckinNotification('daily_enabled', $event.target.checked)"
                      />
                      <span>每日灵修通知</span>
                    </label>
                    <label class="admin-toggle">
                      <input
                        type="checkbox"
                        :checked="checkinNotifications.weekly_enabled !== false"
                        :disabled="!canEditLearning || notificationSaving"
                        @change="setCheckinNotification('weekly_enabled', $event.target.checked)"
                      />
                      <span>周任务通知</span>
                    </label>
                  </div>
                </div>
                <div class="grid cols-2 admin-grid">
                  <div class="card">
                    <h2>每日学习配置</h2>
                    <div class="form-stack admin-form-grid">
                      <label class="admin-toggle"><input type="checkbox" :checked="devotion.enabled !== false" @change="updateLearning(['task_sections','daily','devotion','enabled'], $event.target.checked)" /><span>显示灵修入口</span></label>
                      <label class="admin-field">
                        <span class="admin-field-label">灵修文件</span>
                        <select :value="devotion.path || ''" @change="updateLearning(['task_sections','daily','devotion','path'], $event.target.value)">
                          <option value="">未绑定资源</option>
                          <option v-for="option in markdownOptionsWithCurrent(devotion.path || daily.path)" :key="option.url" :value="option.url">{{ fileOptionText(option) }}</option>
                        </select>
                      </label>
                      <div class="admin-field"><span class="admin-field-label">第 1 篇对应日期</span><DateField :model-value="devotion.numbered_start_date || ''" label="第 1 篇对应日期" @update:model-value="updateLearning(['task_sections','daily','devotion','numbered_start_date'], $event)" /></div>
                      <label class="admin-field"><span class="admin-field-label">起始篇号</span><input type="number" min="1" :value="devotion.numbered_start || 1" @change="updateLearning(['task_sections','daily','devotion','numbered_start'], Number($event.target.value || 1))" /></label>
                      <div class="form-actions"><button :class="canEditLearning ? '' : 'secondary'" :disabled="!canEditLearning" type="button" @click="saveLearningConfig">保存学习配置</button></div>
                    </div>
                  </div>
                  <div class="card">
                    <h2>每日读经配置</h2>
                    <div class="form-stack admin-form-grid">
                      <label class="admin-toggle"><input type="checkbox" :checked="scripture.enabled !== false" @change="updateLearning(['task_sections','daily','scripture','enabled'], $event.target.checked)" /><span>显示每日读经</span></label>
                      <label class="admin-field">
                        <span class="admin-field-label">起始书卷</span>
                        <select :value="scripture.book_id || ''" @change="updateScriptureBook($event.target.value)">
                          <option v-for="book in scriptureBookOptions" :key="book.book_id || book.book" :value="book.book_id">{{ book.book }}（共 {{ book.chapters }} 章）</option>
                        </select>
                      </label>
                      <div class="admin-field"><span class="admin-field-label">读经起始日期</span><DateField :model-value="scripture.start_date || ''" label="读经起始日期" @update:model-value="updateLearning(['task_sections','daily','scripture','start_date'], $event)" /></div>
                      <label class="admin-field"><span class="admin-field-label">起始章</span><input type="number" min="1" :value="scripture.start_chapter || 1" @change="updateLearning(['task_sections','daily','scripture','start_chapter'], Number($event.target.value || 1))" /></label>
                      <div class="form-actions"><button :class="canEditLearning ? '' : 'secondary'" :disabled="!canEditLearning" type="button" @click="saveLearningConfig">保存学习配置</button></div>
                    </div>
                  </div>
                </div>
                <div v-if="weekDraft" class="card week-planner-card">
                  <div class="section-title">
                    <h2>周任务</h2>
                    <div class="inline-actions">
                      <select
                        class="week-picker"
                        :title="weekDraft.id ? weekOptionText(weekDraft) : '新增一周'"
                        :value="weekDraft.id || 0"
                        @change="selectWeekDraft(Number($event.target.value || 0))"
                      >
                        <option v-for="week in weeks" :key="week.id" :value="week.id">{{ weekOptionText(week) }}</option>
                        <option value="0">新增一周</option>
                      </select>
                    </div>
                  </div>
                  <div class="form-stack admin-form-grid">
                    <div class="admin-paired-fields">
                      <div class="admin-field"><span class="admin-field-label">开始时间</span><DateField :model-value="weekDraft.start || ''" label="周任务开始日期" :max="weekDraft.end || ''" @update:model-value="updateWeekDraftField('start', $event)" /></div>
                      <div class="admin-field"><span class="admin-field-label">结束时间</span><DateField :model-value="weekDraft.end || ''" label="周任务结束日期" :min="weekDraft.start || ''" @update:model-value="updateWeekDraftField('end', $event)" /></div>
                    </div>
                    <label class="admin-field">
                      <span class="admin-field-label">自定义标题</span>
                      <input
                        :value="weekDraft.title || ''"
                        maxlength="120"
                        placeholder="留空时根据已选任务内容自动生成"
                        @change="updateWeekDraftField('title', $event.target.value.trim())"
                      />
                      <small class="muted">该标题会显示在任务列表与周任务选择器中。</small>
                    </label>
                    <div class="admin-checkbox-row">
                      <label class="admin-toggle"><input type="checkbox" :checked="enabledFlag(weekDraft.book_enabled)" @change="updateWeekDraftField('book_enabled', $event.target.checked)" /><span>书籍</span></label>
                      <label class="admin-toggle"><input type="checkbox" :checked="enabledFlag(weekDraft.video_enabled)" @change="updateWeekDraftField('video_enabled', $event.target.checked)" /><span>音视频</span></label>
                      <label class="admin-toggle"><input type="checkbox" :checked="enabledFlag(weekDraft.verse_enabled)" @change="updateWeekDraftField('verse_enabled', $event.target.checked)" /><span>背经</span></label>
                      <label class="admin-toggle"><input type="checkbox" :checked="enabledFlag(weekDraft.outline_enabled)" @change="updateWeekDraftField('outline_enabled', $event.target.checked)" /><span>提纲</span></label>
                    </div>
                    <Transition name="admin-task-section">
                      <div v-if="enabledFlag(weekDraft.book_enabled)" class="admin-binding-list">
                        <div class="admin-field-label">读物挂载文件与页码</div>
                        <div v-for="(item, index) in weekDraft.readings || []" :key="`reading-${index}`" class="admin-binding-row reading-binding-row">
                          <select :value="weekBindingSelectionValue(item, readingOptions)" @change="applyBindingSelection('readings', index, $event.target.value)">
                            <option value="">不挂载文件</option>
                            <option v-for="option in readingOptions" :key="librarySelectionValue(option)" :value="librarySelectionValue(option)">{{ optionText(option) }}</option>
                          </select>
                          <div class="admin-page-range">
                            <label class="admin-compact-field">
                              <span>开始页</span>
                              <input type="number" min="1" inputmode="numeric" :value="item.page_start || ''" @change="updateWeekBinding('readings', index, 'page_start', $event.target.value)" />
                            </label>
                            <label class="admin-compact-field">
                              <span>结束页</span>
                              <input type="number" min="1" inputmode="numeric" :value="item.page_end || ''" @change="updateWeekBinding('readings', index, 'page_end', $event.target.value)" />
                            </label>
                          </div>
                          <button class="ghost" type="button" @click="removeWeekBinding('readings', index)">删除</button>
                        </div>
                        <button class="secondary" type="button" @click="addWeekBinding('readings')">新增读物</button>
                      </div>
                    </Transition>
                    <Transition name="admin-task-section">
                      <div v-if="enabledFlag(weekDraft.video_enabled)" class="admin-binding-list">
                        <div class="admin-field-label">音视频文件</div>
                        <div v-for="(item, index) in weekDraft.videos || []" :key="`video-${index}`" class="admin-binding-row video-binding-row">
                          <select :value="weekBindingSelectionValue(item, videoOptions)" @change="applyBindingSelection('videos', index, $event.target.value)">
                            <option value="">不挂载文件</option>
                            <option v-for="option in videoOptions" :key="librarySelectionValue(option)" :value="librarySelectionValue(option)">{{ optionText(option) }}</option>
                          </select>
                          <button class="ghost" type="button" @click="removeWeekBinding('videos', index)">删除</button>
                        </div>
                      </div>
                    </Transition>
                    <Transition name="admin-task-section">
                      <div v-if="enabledFlag(weekDraft.verse_enabled)" class="admin-task-section-fields">
                        <label class="admin-field"><span class="admin-field-label">默写经文</span><input :value="weekDraft.verse_ref || ''" placeholder="例如：罗马书 8:1-5" @change="updateWeekDraftField('verse_ref', $event.target.value)" /></label>
                        <label class="admin-field"><span class="admin-field-label">默写原文</span><textarea rows="4" :value="weekDraft.recite_text || ''" @change="updateWeekDraftField('recite_text', $event.target.value)"></textarea></label>
                      </div>
                    </Transition>
                    <Transition name="admin-task-section">
                      <div v-if="enabledFlag(weekDraft.outline_enabled)" class="admin-binding-list">
                        <div class="admin-field-label">提纲背诵图片</div>
                        <div class="admin-binding-row">
                          <input :value="weekDraft.outline?.title || ''" placeholder="提纲图片标题" @change="updateWeekDraftField('outline', { ...(weekDraft.outline || {}), title: $event.target.value })" />
                          <select :value="librarySelectionValue(weekDraft.outline)" @change="applyOutlineSelection($event.target.value)">
                            <option value="">无提纲图片</option>
                            <option v-for="item in outlineOptions" :key="librarySelectionValue(item)" :value="librarySelectionValue(item)">{{ optionText(item) }}</option>
                          </select>
                        </div>
                      </div>
                    </Transition>
                    <div class="form-actions">
                      <button :disabled="!canEditStudyWeeks" type="button" @click="saveWeekDraft">保存当前周</button>
                      <button class="secondary" :disabled="!canEditStudyWeeks" type="button" @click="restoreWeekDraftDefaults">恢复默认周任务</button>
                      <button class="danger" :disabled="!canEditStudyWeeks" type="button" @click="deleteWeekDraft">删除当前周</button>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section v-else-if="adminSection === 'library'">
              <div class="grid">
                <div class="card">
                  <h2>上传本组资源</h2>
                  <p class="muted">上传后会自动刷新列表，随后即可在“周任务”里选择挂载。</p>
                  <div class="form-stack admin-form-grid">
                    <label class="admin-field">
                      <span class="admin-field-label">上传到</span>
                      <select v-model="uploadCategory">
                        <option v-for="category in RESOURCE_UPLOAD_CATEGORIES" :key="category.key" :value="category.key">{{ category.label }}</option>
                      </select>
                    </label>
                    <label class="admin-field"><span class="admin-field-label">选择文件</span><input ref="uploadInput" type="file" /></label>
                    <div class="form-actions">
                      <button :disabled="!canEditLearning" type="button" @click="uploadSelectedFile">上传到资源库</button>
                      <button class="secondary" type="button" @click="loadAdminData(true)">刷新文件列表</button>
                    </div>
                  </div>
                </div>
                <ResourceGovernance />
              </div>
            </section>

            <section v-else-if="adminSection === 'data'">
              <div class="section-title"><h2>数据导出导入</h2></div>
              <div class="grid cols-2 admin-grid">
                <div class="card">
                  <h2>数据导出</h2>
                  <div class="action-grid">
                    <button type="button" @click="runAdminExport('/admin/exports/checkins-detail', 'checkins-detail.csv', '打卡明细 CSV 已开始下载')">导出打卡明细 CSV</button>
                    <button type="button" @click="runAdminExport('/admin/exports/daily-summary', 'daily-summary.csv', '每日汇总 CSV 已开始下载')">导出每日汇总 CSV</button>
                    <button type="button" @click="runAdminExport('/admin/exports/study-weeks', 'study-weeks.xlsx', '门训任务 Excel 已开始下载')">导出门训任务 Excel</button>
                    <button type="button" @click="runAdminExport('/admin/exports/feedbacks', 'feedbacks.csv', '反馈 CSV 已开始下载')">导出反馈 CSV</button>
                    <button type="button" @click="runAdminExport('/admin/exports/local-backup', 'local-backup.json', '本地备份 JSON 已开始下载')">导出本地备份 JSON</button>
                  </div>
                </div>

                <div class="card">
                  <h2>数据导入</h2>
                  <p class="muted">导入会写入当前小组。门训任务导入会覆盖当前周任务，本地备份导入会恢复当前组数据。</p>
                  <div class="form-stack admin-form-grid">
                    <label class="admin-field">
                      <span class="admin-field-label">导入门训任务 Excel</span>
                      <input ref="studyWeeksImportInput" type="file" accept=".xlsx,.xlsm,.xls" />
                    </label>
                    <div class="form-actions">
                      <button :disabled="!canEditLearning" type="button" @click="runStudyWeeksImport">导入门训任务 Excel</button>
                    </div>
                    <label class="admin-field">
                      <span class="admin-field-label">导入本地备份 JSON</span>
                      <input ref="localBackupImportInput" type="file" accept=".json,application/json" />
                    </label>
                    <div class="form-actions">
                      <button class="danger" :disabled="!canEditLearning" type="button" @click="runLocalBackupImport">导入本地备份 JSON</button>
                    </div>
                  </div>
                </div>
              </div>
            </section>
  </div>
</template>

<style scoped>
.admin-wrapper { min-width: 0; }
.admin-pagehead, .admin-tabs { margin-bottom: 24px; }
.admin-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-width: 100%;
  padding-bottom: 4px;
  overflow: visible;
}
.admin-tabs button { flex: 0 0 auto; min-height: 44px; white-space: nowrap; }
.admin-wrapper :where(input, select, textarea) { max-width: 100%; }
.admin-wrapper :where(.card, .empty) { border-radius: var(--cd-radius-card); }
.admin-wrapper .empty { padding: 32px 20px; text-align: center; }
@media (max-width: 767px) {
  .admin-pagehead { align-items: flex-start; flex-wrap: wrap; gap: 12px; }
  .admin-tabs { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); margin-inline: 0; padding: 6px; }
  .admin-tabs button { width: 100%; white-space: normal; }
  .admin-wrapper :where(button, select, input[type="file"]) { min-height: 44px; }
  .admin-wrapper :where(.form-actions, .inline-actions) { flex-wrap: wrap; }
}
</style>
