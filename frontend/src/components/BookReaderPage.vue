<script setup>
import { defineAsyncComponent, onMounted, ref } from 'vue';
import { RotateCcw, X } from '@lucide/vue';
import { fetchWithAuth, switchGroup } from '../legacy-app';
import { useAppStateStore } from '../stores/appState';

const PdfViewer = defineAsyncComponent(() => import('./PdfViewer.vue'));

const props = defineProps({
  request: {
    type: Object,
    required: true,
  },
});
const appState = useAppStateStore();

const loading = ref(true);
const error = ref('');
const pdfData = ref(null);
const selectedGroupID = ref('');

async function loadBook() {
  loading.value = true;
  error.value = '';
  try {
    if (props.request.groupCode) {
      const group = appState.user?.study_groups?.find((item) => item.code === props.request.groupCode);
      if (!group) throw new Error('当前账号无权阅读该小组的灵修，请使用所属小组账号登录。');
      if (Number(appState.user?.current_group_id) !== Number(group.id)) await switchGroup(group.id);
    } else if (selectedGroupID.value && Number(appState.user?.current_group_id) !== Number(selectedGroupID.value)) {
      await switchGroup(selectedGroupID.value);
    }
    const response = await fetchWithAuth(props.request.sourceURL);
    if (response.status === 404) throw new Error('当前小组未找到这份灵修资料。');
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    pdfData.value = new Uint8Array(await response.arrayBuffer());
  } catch (cause) {
    error.value = cause.message?.startsWith('HTTP ') ? '书籍内容加载失败，请检查网络后重试。' : cause.message;
  } finally {
    loading.value = false;
  }
}

function closeReader() {
  window.close();
  window.setTimeout(() => window.location.assign('/'), 120);
}

onMounted(loadBook);
</script>

<template>
  <main class="standalone-reader">
    <header class="standalone-reader-head">
      <div class="standalone-reader-title">
        <h1>{{ request.title }}</h1>
      </div>
      <button
        class="ghost icon-button standalone-reader-close"
        type="button"
        title="关闭阅读页"
        aria-label="关闭阅读页"
        @click="closeReader"
      >
        <X :size="20" aria-hidden="true" />
      </button>
    </header>

    <section class="standalone-reader-content" aria-live="polite">
      <p v-if="loading" class="muted standalone-reader-status">正在加载书籍内容...</p>
      <div v-else-if="error" class="standalone-reader-error" role="alert">
        <p>{{ error }}</p>
        <label v-if="!request.groupCode && appState.user?.study_groups?.length > 1">
          选择灵修所属小组
          <select v-model="selectedGroupID">
            <option value="">当前小组</option>
            <option v-for="group in appState.user.study_groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option>
          </select>
        </label>
        <button class="secondary icon-text-button" type="button" @click="loadBook">
          <RotateCcw :size="16" aria-hidden="true" />
          重试
        </button>
      </div>
      <PdfViewer
        v-else
        :src="`${request.sourceURL}#page=1`"
        :data="pdfData"
        :title="request.title"
      />
    </section>
  </main>
</template>

<style scoped>
.standalone-reader {
  min-height: 100vh;
  padding: 1rem;
  background: #f5f7f4;
}

.standalone-reader-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  max-width: 1100px;
  margin: 0 auto 1rem;
}

.standalone-reader-title h1 {
  margin: 0;
  font-size: clamp(1.15rem, 4vw, 1.65rem);
  line-height: 1.4;
}

.standalone-reader-content {
  max-width: 1100px;
  margin: auto;
}

.standalone-reader-error {
  display: grid;
  justify-items: start;
  gap: .75rem;
  padding: 1rem;
  background: white;
  border: 1px solid #d8e2d9;
  border-radius: 12px;
}

.standalone-reader-error p { margin: 0; }
.standalone-reader-error label { display: grid; gap: .35rem; }
</style>
