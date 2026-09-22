<script setup>
defineProps({
  items: { type: Array, default: () => [] },
  getKey: { type: Function, required: true },
  getTotal: { type: Function, required: true },
  getHeight: { type: Function, required: true },
  getLabel: { type: Function, required: true },
  emptyLabel: { type: String, default: '暂无打卡数据' },
});
</script>

<template>
  <div v-if="items.length" class="ranking-chart" role="img" aria-label="成员完成数柱状图">
    <div v-for="item in items" :key="getKey(item)" class="ranking-chart__item">
      <small class="ranking-chart__total">{{ getTotal(item) }}</small>
      <div class="ranking-chart__track">
        <span class="ranking-chart__bar" :style="{ height: `${getHeight(item)}%` }"></span>
      </div>
      <span class="ranking-chart__label">{{ getLabel(item) }}</span>
    </div>
  </div>
  <p v-else class="ranking-chart__empty muted small">{{ emptyLabel }}</p>
</template>

<style scoped>
.ranking-chart {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  min-height: 180px;
  padding: 8px 2px 12px;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
  scrollbar-gutter: stable;
}
.ranking-chart__item {
  display: flex;
  flex: 0 0 48px;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.ranking-chart__total {
  color: var(--cd-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
.ranking-chart__track {
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  width: 26px;
  height: 120px;
  overflow: hidden;
  border-radius: 7px;
  background: var(--cd-surface-subtle);
}
.ranking-chart__bar {
  width: 100%;
  min-height: 4px;
  border-radius: 5px 5px 3px 3px;
  background: var(--cd-primary);
  transition: height 0.2s ease;
}
.ranking-chart__label {
  max-width: 48px;
  overflow: hidden;
  color: var(--cd-text);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ranking-chart__empty { padding: 32px 0; text-align: center; }
</style>
