<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { ChevronDown, ChevronUp } from '@lucide/vue';

const props = defineProps({
  items: { type: Array, default: () => [] },
  itemKey: { type: Function, required: true },
  ariaLabel: { type: String, default: '层叠滚动列表' },
  cardHeight: { type: Number, default: 210 },
});

const emit = defineEmits(['change']);
const position = ref(0);
const dragging = ref(false);
let frameID = 0;
let snapTimer = 0;
let startX = 0;
let startY = 0;
let lastX = 0;
let lastTime = 0;
let velocity = 0;
let pointerID = 0;
const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches === true;

const activeIndex = computed(() => props.items.length ? mod(Math.round(position.value), props.items.length) : 0);
const cards = computed(() => {
  const total = props.items.length;
  if (!total) return [];
  const base = Math.floor(position.value);
  const progress = position.value - base;
  return Array.from({ length: Math.min(total, 5) }, (_, slot) => ({
    item: props.items[mod(base + slot, total)],
    slot,
    style: cardStyle(slot, progress),
  }));
});

watch(() => props.items.map((item) => props.itemKey(item)).join('|'), () => {
  stopMotion();
  position.value = 0;
  emitChange();
});

onBeforeUnmount(stopMotion);

function mod(value, total) {
  return ((value % total) + total) % total;
}

function cardStyle(slot, progress) {
  if (slot === 0) {
    const angle = -progress * 88;
    return {
      transform: `translate3d(-50%, calc(-50% + ${reducedMotion ? 0 : progress * -3}px), 0) rotateX(${reducedMotion ? angle * .15 : angle}deg)`,
      opacity: Math.max(0, 1 - Math.pow(Math.max(0, (progress - .3) / .7), 1.4)),
      zIndex: 50,
    };
  }
  const depth = slot - progress;
  return {
    transform: `translate3d(-50%, calc(-50% + ${depth * 5}px), ${-depth * 18}px) scale(${1 - depth * .014})`,
    opacity: Math.max(.42, 1 - depth * .11),
    filter: `brightness(${Math.max(.58, 1 - depth * .075)})`,
    zIndex: 50 - slot,
  };
}

function stopMotion() {
  window.cancelAnimationFrame(frameID);
  window.clearTimeout(snapTimer);
  frameID = 0;
}

function normalize() {
  if (props.items.length && Math.abs(position.value) > 10000) {
    position.value = mod(Math.round(position.value), props.items.length);
  }
}

function emitChange() {
  const item = props.items[activeIndex.value];
  if (item) emit('change', item, activeIndex.value);
}

function snap() {
  stopMotion();
  const from = position.value;
  const to = Math.round(from);
  if (reducedMotion) {
    position.value = to;
    normalize();
    emitChange();
    return;
  }
  const start = performance.now();
  const tick = (now) => {
    const progress = Math.min((now - start) / 240, 1);
    position.value = from + (to - from) * (1 - Math.pow(1 - progress, 4));
    if (progress < 1) {
      frameID = window.requestAnimationFrame(tick);
      return;
    }
    position.value = to;
    normalize();
    emitChange();
  };
  frameID = window.requestAnimationFrame(tick);
}

function inertia() {
  stopMotion();
  if (reducedMotion) return snap();
  let previous = performance.now();
  const tick = (now) => {
    const elapsed = Math.min(32, now - previous);
    previous = now;
    position.value += velocity * elapsed;
    velocity *= Math.pow(.9, elapsed / 16);
    if (Math.abs(velocity) > .00008) {
      frameID = window.requestAnimationFrame(tick);
      return;
    }
    snap();
  };
  frameID = window.requestAnimationFrame(tick);
}

function startDrag(event) {
  if (props.items.length < 2 || event.target.closest('button, a, input, select, textarea, label')) return;
  stopMotion();
  dragging.value = false;
  pointerID = event.pointerId;
  startX = event.clientX;
  startY = event.clientY;
  lastX = event.clientX;
  lastTime = performance.now();
  velocity = 0;
}

function moveDrag(event) {
  if (!pointerID || event.pointerId !== pointerID) return;
  if (!dragging.value) {
    const distanceX = Math.abs(event.clientX - startX);
    const distanceY = Math.abs(event.clientY - startY);
    if (distanceY > distanceX) {
      pointerID = 0;
      return;
    }
    if (distanceX < 10) return;
    dragging.value = true;
    event.currentTarget.setPointerCapture(event.pointerId);
  }
  const now = performance.now();
  const delta = -(event.clientX - lastX) / Math.max(180, event.currentTarget.clientWidth * .72);
  velocity = delta / Math.max(1, now - lastTime);
  position.value += delta;
  lastX = event.clientX;
  lastTime = now;
}

function endDrag() {
  pointerID = 0;
  if (!dragging.value) return;
  dragging.value = false;
  inertia();
}

function onWheel(event) {
  if (props.items.length < 2 || (Math.abs(event.deltaX) <= Math.abs(event.deltaY) && !event.shiftKey)) return;
  event.preventDefault();
  stopMotion();
  const delta = event.shiftKey ? event.deltaY : event.deltaX;
  position.value += Math.max(-90, Math.min(90, delta)) * .0035;
  snapTimer = window.setTimeout(snap, 90);
}

function onKey(event) {
  if (!['ArrowUp', 'ArrowDown'].includes(event.key) || props.items.length < 2) return;
  event.preventDefault();
  stopMotion();
  position.value += event.key === 'ArrowDown' ? 1 : -1;
  snap();
}

function step(offset) {
  if (props.items.length < 2) return;
  stopMotion();
  position.value = Math.round(position.value) + offset;
  snap();
}
</script>

<template>
  <section class="stacked-wheel" :style="{ '--stack-card-height': `${cardHeight}px` }">
    <div class="stacked-wheel__head">
      <span>{{ ariaLabel }}</span>
      <div v-if="items.length" class="stacked-wheel__controls">
        <button class="quiet" type="button" :aria-label="`上一项${ariaLabel}`" @click="step(-1)"><ChevronUp :size="16" /></button>
        <b>{{ String(activeIndex + 1).padStart(2, '0') }} / {{ String(items.length).padStart(2, '0') }}</b>
        <button class="quiet" type="button" :aria-label="`下一项${ariaLabel}`" @click="step(1)"><ChevronDown :size="16" /></button>
      </div>
    </div>
    <div
      v-if="items.length"
      class="stacked-wheel__stage"
      :class="{ dragging }"
      role="region"
      :aria-label="ariaLabel"
      tabindex="0"
      @pointerdown="startDrag"
      @pointermove="moveDrag"
      @pointerup="endDrag"
      @pointercancel="endDrag"
      @lostpointercapture="endDrag"
      @wheel="onWheel"
      @keydown="onKey"
    >
      <article
        v-for="card in cards"
        :key="card.slot"
        class="stacked-wheel__card"
        :class="{ active: card.slot === 0 }"
        :style="card.style"
        role="group"
        :aria-label="`${ariaLabel}第 ${activeIndex + 1} 项`"
        :aria-hidden="card.slot !== 0"
        :inert="card.slot !== 0"
      >
        <slot :item="card.item" :active="card.slot === 0" />
      </article>
    </div>
    <div v-else class="stacked-wheel__empty"><slot name="empty">暂无内容</slot></div>
  </section>
</template>

<style scoped>
.stacked-wheel { min-width: 0; }
.stacked-wheel__head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 8px; color: var(--cd-muted); font-size: 12px; }
.stacked-wheel__head b { color: var(--cd-text); font-variant-numeric: tabular-nums; }
.stacked-wheel__controls { display: flex; align-items: center; gap: 4px; }
.stacked-wheel__controls button { display: grid; width: 32px; min-width: 32px; min-height: 32px; padding: 0; place-items: center; }
.stacked-wheel__stage { position: relative; height: calc(var(--stack-card-height) + 42px); overflow: hidden; perspective: 900px; transform-style: preserve-3d; touch-action: pan-y; user-select: none; cursor: grab; outline: none; }
.stacked-wheel__stage:focus-visible { border-radius: var(--cd-radius-card); box-shadow: 0 0 0 3px rgb(47 107 69 / 18%); }
.stacked-wheel__stage.dragging { cursor: grabbing; }
.stacked-wheel__card { position: absolute; top: 50%; left: 50%; width: calc(100% - 10px); height: var(--stack-card-height); overflow: hidden; transform-origin: 50% 50% -140px; transform-style: preserve-3d; backface-visibility: hidden; will-change: transform, opacity, filter; }
.stacked-wheel__card:not(.active), .stacked-wheel__stage.dragging .stacked-wheel__card { pointer-events: none; }
.stacked-wheel__card > :deep(*) { height: 100%; }
.stacked-wheel__empty { display: grid; min-height: 112px; place-items: center; border: 1px dashed var(--cd-border); border-radius: var(--cd-radius-card); color: var(--cd-muted); font-size: 13px; }
@media (prefers-reduced-motion: reduce) { .stacked-wheel__card { will-change: auto; } }
</style>
