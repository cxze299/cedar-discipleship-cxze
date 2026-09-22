<script setup>
import { LogOut } from '@lucide/vue';
import GroupSwitcher from './GroupSwitcher.vue';

defineProps({
  pageTitle: { type: String, required: true },
  groups: { type: Array, default: () => [] },
  currentGroupID: { type: [String, Number], default: '' },
  defaultGroupID: { type: [String, Number], default: '' },
  activeGroup: { type: Object, default: null },
});
defineEmits(['switch-group', 'set-default', 'logout']);
</script>

<template>
  <header class="topbar app-page-header">
    <div class="inline app-page-header__title">
      <span class="trail">学习空间 <span class="app-page-header__separator">/</span><span class="crumb-active">{{ pageTitle }}</span></span>
      <span class="mobiletitle">{{ pageTitle }}</span>
    </div>
    <div class="inline topbar-actions app-page-header__actions">
      <GroupSwitcher
        :groups="groups"
        :current-group-i-d="currentGroupID"
        :default-group-i-d="defaultGroupID"
        :active-group="activeGroup"
        @switch="$emit('switch-group', $event)"
        @set-default="$emit('set-default', $event)"
      />
      <button class="quiet topbar-logout app-page-header__logout" type="button" title="退出登录" @click="$emit('logout')">
        <LogOut :size="16" />
        <span>退出</span>
      </button>
    </div>
  </header>
</template>
