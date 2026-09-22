<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import {
  CalendarDays,
  Check,
  Download,
  LoaderCircle,
  Plus,
  Save,
  X,
} from '@lucide/vue';
import { api, fetchWithAuth, toast as showToast } from '../legacy-app';
import DateField from './ui/DateField.vue';

const props = defineProps({
  groupId: {
    type: Number,
    required: true,
  },
  active: {
    type: Boolean,
    default: false,
  },
});

const weekdayOptions = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 7, label: '周日' },
];

const month = ref(currentMonth());
const sheet = ref(null);
const loading = ref(false);
const saving = ref(false);
const weekdays = ref([]);
const extraDates = ref([]);
const extraDateDraft = ref('');
const sortBy = ref('name');
const sortDirection = ref('asc');

const sortedMembers = computed(() => {
  const members = [...(sheet.value?.members || [])];
  const direction = sortDirection.value === 'asc' ? 1 : -1;
  return members.sort((left, right) => {
    if (sortBy.value === 'count' && left.present_count !== right.present_count) {
      return (left.present_count - right.present_count) * direction;
    }
    return String(left.display_name).localeCompare(String(right.display_name), 'zh-CN') * direction;
  });
});

watch(
  () => [props.active, props.groupId],
  async ([active]) => {
    if (active) await loadAttendance();
  },
);

onMounted(async () => {
  if (props.active) await loadAttendance();
});

async function loadAttendance() {
  loading.value = true;
  try {
    sheet.value = await api(`/ministry-groups/${props.groupId}/attendance?month=${month.value}`);
    weekdays.value = [...(sheet.value.settings?.weekdays || [])];
    extraDates.value = [...(sheet.value.settings?.extra_dates || [])];
  } catch (error) {
    sheet.value = null;
    showToast(error.message);
  } finally {
    loading.value = false;
  }
}

async function shiftMonth(offset) {
  const [year, monthNumber] = month.value.split('-').map(Number);
  const next = new Date(year, monthNumber - 1 + offset, 1);
  month.value = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}`;
  await loadAttendance();
}

function toggleWeekday(weekday) {
  weekdays.value = weekdays.value.includes(weekday)
    ? weekdays.value.filter((value) => value !== weekday)
    : [...weekdays.value, weekday].sort((left, right) => left - right);
}

function addExtraDate() {
  if (!extraDateDraft.value || extraDates.value.includes(extraDateDraft.value)) return;
  extraDates.value = [...extraDates.value, extraDateDraft.value].sort();
  extraDateDraft.value = '';
}

function removeExtraDate(date) {
  extraDates.value = extraDates.value.filter((value) => value !== date);
}

async function saveSettings() {
  saving.value = true;
  try {
    await api(`/ministry-groups/${props.groupId}/attendance/settings`, {
      method: 'PUT',
      body: JSON.stringify({
        weekdays: weekdays.value,
        extra_dates: extraDates.value,
      }),
    });
    showToast('考勤日期设置已保存');
    await loadAttendance();
  } catch (error) {
    showToast(error.message);
  } finally {
    saving.value = false;
  }
}

async function toggleAttendance(member, date) {
  if (!sheet.value?.can_mark || saving.value) return;
  const present = !member.present?.[date];
  saving.value = true;
  try {
    await api(`/ministry-groups/${props.groupId}/attendance/${date}/members/${member.user_id}`, {
      method: 'PUT',
      body: JSON.stringify({ present }),
    });
    member.present = { ...(member.present || {}), [date]: present };
    if (!present) delete member.present[date];
    member.present_count = Object.values(member.present).filter(Boolean).length;
  } catch (error) {
    showToast(error.message);
  } finally {
    saving.value = false;
  }
}

async function exportAttendance() {
  try {
    const response = await fetchWithAuth(`/api/ministry-groups/${props.groupId}/attendance/export?month=${month.value}`);
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || `HTTP ${response.status}`);
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `数点组考勤-${month.value}.csv`;
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch (error) {
    showToast(error.message);
  }
}

function dateLabel(date) {
  const value = new Date(`${date}T00:00:00`);
  return `${value.getMonth() + 1}/${value.getDate()} ${weekdayOptions[(value.getDay() + 6) % 7].label}`;
}

function currentMonth() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}
</script>

<template>
  <section class="attendance-workspace">
    <div class="attendance-toolbar">
      <div>
        <div class="eyebrow">数点与考勤</div>
        <h3>月度考勤表</h3>
      </div>
      <div class="attendance-month">
        <button class="secondary icon-button" type="button" title="上个月" aria-label="查看上个月" @click="shiftMonth(-1)">‹</button>
        <DateField v-model="month" mode="month" label="选择考勤月份" @change="loadAttendance" />
        <button class="secondary icon-button" type="button" title="下个月" aria-label="查看下个月" @click="shiftMonth(1)">›</button>
        <button class="secondary icon-text-button" type="button" @click="exportAttendance">
          <Download :size="16" /> 导出 CSV
        </button>
      </div>
    </div>

    <div v-if="loading" class="ministry-loading">
      <LoaderCircle :size="22" class="spin" /> 正在加载考勤表
    </div>

    <template v-else-if="sheet">
      <section v-if="sheet.can_manage" class="attendance-settings">
        <div class="attendance-settings-head">
          <div>
            <h4>固定考勤日</h4>
          </div>
          <button class="icon-text-button" type="button" :disabled="saving" @click="saveSettings">
            <Save :size="16" /> 保存设置
          </button>
        </div>
        <div class="weekday-picker">
          <label v-for="weekday in weekdayOptions" :key="weekday.value">
            <input
              type="checkbox"
              :checked="weekdays.includes(weekday.value)"
              @change="toggleWeekday(weekday.value)"
            />
            <span>{{ weekday.label }}</span>
          </label>
        </div>
        <div class="extra-date-editor">
          <div>
            <span>额外考勤日期</span>
            <span class="extra-date-input">
              <DateField v-model="extraDateDraft" label="选择额外考勤日期" />
              <button class="secondary icon-button" type="button" title="添加日期" aria-label="添加额外考勤日期" @click="addExtraDate">
                <Plus :size="16" />
              </button>
            </span>
          </div>
          <div class="extra-date-list">
            <span v-for="date in extraDates" :key="date">
              <CalendarDays :size="14" /> {{ date }}
              <button type="button" title="移除日期" :aria-label="`移除考勤日期${date}`" @click="removeExtraDate(date)"><X :size="13" /></button>
            </span>
            <small v-if="!extraDates.length">暂无额外日期</small>
          </div>
        </div>
      </section>

      <div class="attendance-table-tools">
        <div>
          <strong>{{ sheet.dates.length }} 个考勤日</strong>
          <span>{{ sheet.members.length }} 位学习小组成员</span>
        </div>
        <div class="attendance-sort">
          <select v-model="sortBy">
            <option value="name">按姓名</option>
            <option value="count">按出勤次数</option>
          </select>
          <button class="secondary" type="button" @click="sortDirection = sortDirection === 'asc' ? 'desc' : 'asc'">
            {{ sortDirection === 'asc' ? '升序' : '降序' }}
          </button>
        </div>
      </div>

      <div v-if="sheet.dates.length" class="attendance-table-scroll">
        <table class="attendance-table">
          <thead>
            <tr>
              <th>成员</th>
              <th v-for="date in sheet.dates" :key="date">{{ dateLabel(date) }}</th>
              <th>出勤</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="member in sortedMembers" :key="member.user_id">
              <td>
                <b>{{ member.display_name }}</b>
                <small>{{ member.username }}</small>
              </td>
              <td v-for="date in sheet.dates" :key="`${member.user_id}:${date}`">
                <button
                  class="attendance-cell"
                  :class="{ present: member.present?.[date] }"
                  type="button"
                  :disabled="!sheet.can_mark || saving"
                  :title="`${member.display_name} · ${date}`"
                  @click="toggleAttendance(member, date)"
                >
                  <Check v-if="member.present?.[date]" :size="17" />
                  <span v-else></span>
                </button>
              </td>
              <td><strong>{{ member.present_count }}/{{ sheet.dates.length }}</strong></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty">本月没有需要考勤的日期，请由管理员设置固定星期或额外日期。</div>
    </template>
  </section>
</template>

<style scoped>
.attendance-workspace { min-width: 0; }
.attendance-toolbar, .attendance-month, .attendance-settings-head, .attendance-table-tools { gap: 12px; }
.attendance-month :where(button, input),
.attendance-settings button,
.attendance-sort :where(button, select) { min-height: 44px; }
.attendance-month .icon-button, .extra-date-input .icon-button { min-width: 44px; }
.extra-date-list button { display: inline-grid; place-items: center; width: 44px; min-width: 44px; height: 44px; padding: 0; }
.attendance-table-scroll { max-width: 100%; overflow-x: auto; overscroll-behavior-inline: contain; scrollbar-gutter: stable; }
.attendance-table { width: max-content; min-width: 100%; }
.attendance-cell { display: inline-grid; place-items: center; min-width: 44px; min-height: 44px; }
.ministry-loading, .empty { padding: 32px 20px; text-align: center; }
@media (max-width: 767px) {
  .attendance-toolbar, .attendance-settings-head, .attendance-table-tools { align-items: stretch; flex-direction: column; }
  .attendance-month { display: grid; grid-template-columns: 44px minmax(0, 1fr) 44px; width: 100%; }
  .attendance-month .icon-text-button { grid-column: 1 / -1; justify-content: center; }
  .weekday-picker { grid-template-columns: repeat(4, minmax(0, 1fr)); }
  .weekday-picker label { min-height: 44px; }
  .extra-date-input { display: grid; grid-template-columns: minmax(0, 1fr) 44px; }
  .attendance-sort { display: grid; grid-template-columns: minmax(0, 1fr) auto; width: 100%; }
  .attendance-table-scroll { margin-inline: -16px; padding-inline: 16px; }
  .attendance-table th:first-child,
  .attendance-table td:first-child {
    position: sticky;
    left: 0;
    z-index: 1;
    min-width: 136px;
    background: var(--cd-surface, #fff);
    box-shadow: 1px 0 0 var(--cd-border);
  }
  .attendance-table thead th:first-child { z-index: 2; background: var(--cd-surface-subtle); }
}
</style>
