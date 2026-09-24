<script setup>
import { nextTick, ref, watch } from 'vue';
import {
  AlertCircle,
  CheckCircle2,
  Info,
  X,
} from '@lucide/vue';
import AppOverlay from './ui/AppOverlay.vue';
import { dialogState, resolveDialog } from '../ui/dialog';

const input = ref(null);

watch(() => dialogState.open, async (open) => {
  if (open) {
    await nextTick();
    if (dialogState.mode === 'prompt') {
      input.value?.focus();
      input.value?.select();
    }
  }
});
</script>

<template>
  <AppOverlay
    :open="dialogState.open"
    variant="modal"
    title-id="site-dialog-title"
    panel-class="cd-dialog"
    header-class="cd-dialog-head"
    body-class="cd-dialog-body"
    footer-class="cd-dialog-foot"
    @close="resolveDialog(false)"
  >
    <template #header>
          <div class="inline site-dialog-title">
            <component
              :is="
                dialogState.tone === 'danger'
                  ? AlertCircle
                  : dialogState.tone === 'success'
                  ? CheckCircle2
                  : Info
              "
              :size="20"
              :style="{
                color:
                  dialogState.tone === 'danger'
                    ? 'var(--cd-danger)'
                    : dialogState.tone === 'success'
                    ? 'var(--cd-success)'
                    : 'var(--cd-primary)',
              }"
            />
            <h2 id="site-dialog-title">{{ dialogState.title }}</h2>
          </div>
          <button
            class="cd-dialog-close"
            type="button"
            aria-label="关闭窗口"
            @click="resolveDialog(false)"
          >
            <X :size="20" />
          </button>
    </template>

    <template #default>
          <p v-if="dialogState.message" style="margin: 0 0 14px 0; color: var(--cd-muted);">
            {{ dialogState.message }}
          </p>
          <div v-if="dialogState.mode === 'prompt'" style="margin-top: 10px;">
            <input
              ref="input"
              data-dialog-autofocus
              :aria-label="dialogState.title"
              v-model="dialogState.value"
              style="width: 100%;"
              :placeholder="dialogState.placeholder"
              @keydown.enter.prevent="resolveDialog(true)"
            />
          </div>
    </template>

    <template #footer>
          <button
            v-if="dialogState.mode !== 'alert'"
            type="button"
            @click="resolveDialog(false)"
          >
            {{ dialogState.cancelLabel || '取消' }}
          </button>
          <button
            :class="dialogState.tone === 'danger' ? 'danger' : 'primary'"
            type="button"
            @click="resolveDialog(true)"
          >
            {{ dialogState.confirmLabel || '确认' }}
          </button>
    </template>
  </AppOverlay>
</template>

<style scoped>
:global(.cd-dialog-head) {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) 40px;
  align-items: center;
}
.site-dialog-title {
  grid-column: 2;
  justify-self: center;
  min-width: 0;
  text-align: center;
}
.site-dialog-title h2 { overflow-wrap: anywhere; }
.cd-dialog-close {
  grid-column: 3;
  justify-self: end;
  width: 40px;
  min-width: 40px;
  padding: 0;
}
</style>
