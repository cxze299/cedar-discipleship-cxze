<script setup>
defineProps({ error: { type: Error, default: null } });
function reload() {
  window.location.reload();
}
</script>

<template>
  <div class="empty" :role="error ? 'alert' : 'status'" :aria-busy="!error">
    <p>{{ error ? '页面加载失败，请检查网络后重试。' : '正在加载页面…' }}</p>
    <button v-if="error" class="primary" type="button" @click="reload">重新加载</button>
  </div>
</template>
