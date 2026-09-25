<script setup>
import { vDialogFocus } from '../../ui/dialogFocus';

const props = defineProps({
  open: { type: Boolean, default: false },
  variant: {
    type: String,
    default: 'modal',
    validator: (value) => ['modal', 'drawer', 'viewer'].includes(value),
  },
  title: { type: String, default: '' },
  panelId: { type: String, default: '' },
  titleId: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  closeLabel: { type: String, default: '关闭窗口' },
  dismissible: { type: Boolean, default: true },
  panelClass: { type: [String, Array, Object], default: '' },
  headerClass: { type: [String, Array, Object], default: '' },
  bodyClass: { type: [String, Array, Object], default: '' },
  footerClass: { type: [String, Array, Object], default: '' },
});

const emit = defineEmits(['close']);

function close(reason) {
  if (!props.dismissible) return;
  emit('close', reason);
}
</script>

<template>
  <Teleport to="body">
    <Transition name="app-overlay">
      <div
        v-if="open"
        class="app-overlay"
        :class="`app-overlay--${variant}`"
        @pointerdown.self="close('backdrop')"
      >
        <section
          v-dialog-focus="() => close('escape')"
          :id="panelId || undefined"
          class="app-overlay__panel"
          :class="panelClass"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId || undefined"
          :aria-label="titleId ? undefined : (ariaLabel || title || '对话窗口')"
        >
          <header
            v-if="$slots.header || title"
            class="app-overlay__header"
            :class="[$slots.header ? 'app-overlay__header--custom' : 'app-overlay__header--title', headerClass]"
          >
            <slot name="header" :close="close">
              <h2 :id="titleId || undefined">{{ title }}</h2>
              <button type="button" class="app-overlay__close" :aria-label="closeLabel" @click="close('button')">
                <span aria-hidden="true">×</span>
              </button>
            </slot>
          </header>

          <div class="app-overlay__body" :class="bodyClass">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="app-overlay__footer" :class="footerClass">
            <slot name="footer" :close="close" />
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.app-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--cd-z-overlay, 1000);
  display: flex;
  padding: var(--cd-space-6, 24px);
  background: var(--cd-backdrop, rgb(22 34 27 / 38%));
  overscroll-behavior: contain;
}

.app-overlay--modal,
.app-overlay--viewer {
  align-items: center;
  justify-content: center;
}

.app-overlay--modal,
.app-overlay--viewer {
  z-index: var(--cd-z-dialog, 1100);
}

.app-overlay--drawer {
  justify-content: flex-end;
  padding: 0;
}

.app-overlay__panel {
  position: relative;
  inset: auto;
  display: flex;
  flex-direction: column;
  min-width: 0;
  max-height: calc(100dvh - (var(--cd-space-6, 24px) * 2));
  overflow: hidden;
  border: 1px solid var(--cd-border);
  border-radius: var(--cd-radius-dialog);
  background: var(--cd-surface);
  box-shadow: var(--cd-shadow-dialog);
  outline: none;
}

.app-overlay--modal .app-overlay__panel {
  width: min(30rem, 100%);
}

.app-overlay--drawer .app-overlay__panel {
  width: min(30rem, 100%);
  height: 100dvh;
  max-height: 100dvh;
  border-width: 0 0 0 1px;
  border-radius: 0;
}

.app-overlay--viewer .app-overlay__panel {
  width: min(1100px, 100%);
  height: min(960px, calc(100dvh - (var(--cd-space-6, 24px) * 2)));
}

.app-overlay__header,
.app-overlay__footer {
  flex: 0 0 auto;
}

.app-overlay__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--cd-space-4, 16px);
  padding: var(--cd-space-5, 20px) var(--cd-space-6, 24px);
  border-bottom: 1px solid var(--cd-border);
}

.app-overlay__header h2 {
  min-width: 0;
  overflow-wrap: anywhere;
}

.app-overlay__header--title {
  display: grid;
  grid-template-columns: var(--cd-control-h-desktop, 40px) minmax(0, 1fr) var(--cd-control-h-desktop, 40px);
  text-align: center;
}

.app-overlay__header--title h2 {
  grid-column: 2;
}

.app-overlay__header--title .app-overlay__close {
  grid-column: 3;
  justify-self: end;
}

.app-overlay__body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: var(--cd-space-6, 24px);
}

.app-overlay__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--cd-space-3, 12px);
  padding: var(--cd-space-4, 16px) var(--cd-space-6, 24px);
  padding-bottom: max(var(--cd-space-4, 16px), env(safe-area-inset-bottom));
  border-top: 1px solid var(--cd-border);
}

.app-overlay__close {
  width: var(--cd-control-h-desktop, 40px);
  min-width: var(--cd-control-h-desktop, 40px);
  padding: 0;
  border-color: transparent;
  background: transparent;
  font-size: 1.5rem;
  line-height: 1;
}

.app-overlay-enter-active,
.app-overlay-leave-active {
  transition: opacity 160ms ease;
}

.app-overlay-enter-active .app-overlay__panel,
.app-overlay-leave-active .app-overlay__panel {
  transition: transform 180ms ease, opacity 160ms ease;
}

.app-overlay-enter-from,
.app-overlay-leave-to {
  opacity: 0;
}

.app-overlay--drawer.app-overlay-enter-from .app-overlay__panel,
.app-overlay--drawer.app-overlay-leave-to .app-overlay__panel {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .app-overlay {
    padding: 0;
  }

  .app-overlay--modal {
    padding: var(--cd-space-5, 20px);
  }

  .app-overlay--modal .app-overlay__panel {
    width: 100%;
    height: auto;
    max-height: 90dvh;
    border: 1px solid var(--cd-border);
    border-radius: var(--cd-radius-dialog);
  }

  .app-overlay--viewer .app-overlay__panel {
    width: 100%;
    height: 100dvh;
    max-height: 100dvh;
    border: 0;
    border-radius: 0;
  }

  .app-overlay__header {
    padding: max(var(--cd-space-4, 16px), env(safe-area-inset-top)) var(--cd-space-4, 16px) var(--cd-space-4, 16px);
  }

  .app-overlay__body {
    padding: var(--cd-space-4, 16px);
  }

  .app-overlay__footer {
    padding-inline: var(--cd-space-4, 16px);
  }
}
</style>
