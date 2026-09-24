<script setup>
import { BarChart2, Book, Folder, MoreHorizontal, Users } from '@lucide/vue';

defineProps({ tab: { type: String, required: true }, moreOpen: Boolean });
defineEmits(['navigate', 'more']);

const items = [
  ['home', '学习', Book],
  ['dashboard', '统计', BarChart2],
  ['groups', '小组', Users],
  ['resources', '资料', Folder],
];
</script>

<template>
  <nav class="mobilebar app-mobile-nav" aria-label="手机主导航">
    <button
      v-for="item in items"
      :key="item[0]"
      :class="{ active: tab === item[0] }"
      :aria-current="tab === item[0] ? 'page' : undefined"
      type="button"
      @click="$emit('navigate', item[0])"
    >
      <component :is="item[2]" :size="20" stroke-width="1.8" />
      <span>{{ item[1] }}</span>
    </button>
    <button type="button" aria-haspopup="dialog" :aria-expanded="moreOpen" @click="$emit('more')">
      <MoreHorizontal :size="20" stroke-width="1.8" />
      <span>更多</span>
    </button>
  </nav>
</template>
