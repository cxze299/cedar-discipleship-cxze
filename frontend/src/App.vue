<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import AppRoot from './components/AppRoot.vue';
import BookReaderPage from './components/BookReaderPage.vue';
import CheckinWorkbench from './components/CheckinWorkbench.vue';
import ContentViewer from './components/ContentViewer.vue';
import Dashboard from './components/Dashboard.vue';
import DownloadCenter from './components/DownloadCenter.vue';
import MinistryGroups from './components/MinistryGroups.vue';
import SiteDialog from './components/SiteDialog.vue';
import AppStatus from './components/ui/AppStatus.vue';
import AppToast from './components/ui/AppToast.vue';
import { useAppShellStore } from './stores/appShell';
import { useAppStateStore } from './stores/appState';
import { disposeApp, initializeApp } from './legacy-app';
import { parseReaderPageRequest } from './runtime/content';
import { readComponentTheme } from './styles/theme';

const componentTheme = readComponentTheme();

const shell = useAppShellStore();
const appState = useAppStateStore();
const readerRequest = parseReaderPageRequest(window.location.search);
const retrying = ref(false);

async function boot() {
  if (retrying.value) return;
  retrying.value = true;
  shell.setMounting();
  try {
    if (readerRequest) {
      shell.setReady();
      return;
    }
    await initializeApp();
    shell.setReady();
  } catch (error) {
    shell.setError(error);
  } finally {
    retrying.value = false;
  }
}

async function retryBoot() {
  disposeApp();
  await boot();
}

onMounted(boot);

onBeforeUnmount(() => {
  disposeApp();
});
</script>

<template>
  <a-config-provider :theme="componentTheme">
  <main class="antd-app-shell" :data-status="shell.status">
    <BookReaderPage v-if="readerRequest" :request="readerRequest" />
    <AppStatus
      v-else-if="shell.error"
      status="error"
      title="前端加载失败"
      :description="String(shell.error || '网络或数据服务异常')"
    >
      <template #action>
        <a-button type="primary" :loading="retrying" @click="retryBoot">
          重新加载
        </a-button>
      </template>
    </AppStatus>
    <AppStatus
      v-else-if="shell.status !== 'ready'"
      status="loading"
      title="正在载入"
      description="正在载入门训打卡数据..."
    />
    <template v-else>
      <AppRoot />
      <template v-if="appState.authenticated">
        <CheckinWorkbench />
        <Dashboard />
        <MinistryGroups />
        <ContentViewer />
        <DownloadCenter />
      </template>
    </template>
    <SiteDialog />
    <AppToast :message="appState.toast" />
  </main>
  </a-config-provider>
</template>
