<script setup>
defineProps({
  message: {
    type: String,
    default: '',
  },
});
</script>

<template>
  <Transition name="app-toast">
    <div
      v-if="message"
      class="app-toast"
      role="status"
      aria-live="polite"
      aria-atomic="true"
    >
      {{ message }}
    </div>
  </Transition>
</template>

<style scoped>
.app-toast {
  position: fixed;
  z-index: var(--cd-z-toast);
  left: 50%;
  bottom: max(var(--cd-space-8), calc(env(safe-area-inset-bottom) + var(--cd-space-4)));
  transform: translateX(-50%);
  width: max-content;
  max-width: min(32rem, calc(100vw - (var(--cd-space-4) * 2)));
  padding: var(--cd-space-3) var(--cd-space-4);
  border: 1px solid rgb(255 255 255 / 18%);
  border-radius: var(--cd-radius-card);
  color: var(--cd-surface);
  background: rgb(32 43 37 / 94%);
  box-shadow: var(--cd-shadow-floating);
  font-size: var(--cd-font-size-body);
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.app-toast-enter-active,
.app-toast-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.app-toast-enter-from,
.app-toast-leave-to {
  opacity: 0;
  transform: translate(-50%, var(--cd-space-2));
}

@media (max-width: 767px) {
  .app-toast {
    bottom: max(var(--cd-space-6), calc(env(safe-area-inset-bottom) + var(--cd-space-4)));
  }
}
</style>
