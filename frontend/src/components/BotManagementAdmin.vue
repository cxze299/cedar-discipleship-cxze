<script setup>
import { onMounted, ref } from 'vue';
import { Bot, CircleAlert, CircleCheck, KeyRound, Plus, RefreshCw } from '@lucide/vue';
import { api, toast as showToast } from '../legacy-app';

const configured = ref(false);
const robots = ref([]);
const studyGroups = ref([]);
const loading = ref(false);
const savingBinding = ref('');
const savingRobot = ref(false);
const newRobot = ref({ id: '', name: '', token: '' });

onMounted(load);

async function load() {
  loading.value = true;
  try {
    const result = await api('/super-admin/bot-management');
    configured.value = result.configured === true;
    robots.value = result.robots || [];
    studyGroups.value = result.study_groups || [];
  } catch (error) {
    showToast(error.message);
  } finally {
    loading.value = false;
  }
}

async function assign(robot, chat, event) {
  const groupID = Number(event.target.value || 0);
  const previousGroupID = Number(chat.group_id || 0);
  chat.group_id = groupID;
  savingBinding.value = `${robot.id}:${chat.chat_id}`;
  try {
    await api('/super-admin/bot-bindings', {
      method: 'PUT',
      body: JSON.stringify({
        robot_id: robot.id,
        chat_id: chat.chat_id,
        chat_type: chat.chat_type,
        group_id: groupID,
      }),
    });
    showToast(groupID ? '群聊绑定已保存' : '群聊绑定已取消');
    await load();
  } catch (error) {
    chat.group_id = previousGroupID;
    showToast({
      bot_chat_not_found: '机器人已不在该群聊中',
      robot_not_found: '机器人配置不存在',
      robot_authentication_failed: '机器人认证失败',
      study_group_not_found: '学习小组不存在',
      bot_binding_save_failed: '群聊绑定保存失败',
    }[error.message] || error.message);
  } finally {
    savingBinding.value = '';
  }
}

async function createRobot() {
  const payload = {
    id: newRobot.value.id.trim(),
    name: newRobot.value.name.trim(),
    token: newRobot.value.token.trim(),
  };
  if (!payload.token) {
    showToast('请输入机器人 Token');
    return;
  }
  savingRobot.value = true;
  try {
    await api('/super-admin/bot-robots', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    newRobot.value = { id: '', name: '', token: '' };
    showToast('机器人已新增');
    await load();
  } catch (error) {
    showToast({
      invalid_robot_config: '机器人配置无效',
      robot_authentication_failed: '机器人认证失败',
      robot_already_exists: '机器人 ID 已存在',
      robot_token_exists: '机器人 Token 已存在',
      robot_limit_exceeded: '机器人数量已达上限',
      bot_robot_save_failed: '机器人保存失败',
    }[error.message] || error.message);
  } finally {
    savingRobot.value = false;
  }
}

function checkedAt(value) {
  if (!value) return '';
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}
</script>

<template>
  <section>
    <div class="section-title bot-management-title">
      <h2>机器人管理</h2>
      <button
        class="secondary icon-button"
        type="button"
        title="刷新群聊"
        :disabled="loading"
        @click="load"
      >
        <RefreshCw :size="16" :class="{ spinning: loading }" />
      </button>
    </div>

    <form class="card bot-registration" @submit.prevent="createRobot">
      <div class="bot-registration-heading">
        <span class="bot-chat-icon"><KeyRound :size="18" /></span>
        <strong>新增机器人</strong>
      </div>
      <div class="bot-registration-grid">
        <label class="admin-field">
          <span class="admin-field-label">机器人 Token</span>
          <input
            v-model="newRobot.token"
            type="password"
            autocomplete="off"
            placeholder="123:secret"
            :disabled="savingRobot"
          >
        </label>
        <label class="admin-field">
          <span class="admin-field-label">显示名称</span>
          <input
            v-model="newRobot.name"
            autocomplete="off"
            placeholder="主机器人"
            :disabled="savingRobot"
          >
        </label>
        <label class="admin-field">
          <span class="admin-field-label">机器人 ID</span>
          <input
            v-model="newRobot.id"
            autocomplete="off"
            placeholder="primary"
            :disabled="savingRobot"
          >
        </label>
        <button class="ok bot-registration-button" type="submit" :disabled="savingRobot">
          <Plus :size="16" />
          新增
        </button>
      </div>
    </form>

    <div v-if="loading && !robots.length" class="empty">正在读取机器人状态…</div>
    <div v-else-if="!configured" class="empty">暂无机器人</div>
    <div v-else class="card bot-management">
      <section v-for="robot in robots" :key="robot.id" class="bot-robot-section">
        <header class="bot-robot-header">
          <div class="bot-chat-main">
            <span class="bot-chat-icon"><Bot :size="18" /></span>
            <div>
              <strong>{{ robot.name }}</strong>
              <div class="muted">
                {{ robot.identity?.username ? `@${robot.identity.username}` : robot.id }}
              </div>
            </div>
          </div>
          <div class="bot-robot-status" :class="`is-${robot.state}`">
            <CircleCheck v-if="robot.state === 'healthy'" :size="16" />
            <CircleAlert v-else :size="16" />
            <span>{{ robot.state === 'healthy' ? '运行正常' : robot.state === 'degraded' ? '部分异常' : '不可用' }}</span>
          </div>
          <div class="bot-robot-metrics muted">
            待发送 {{ robot.queue?.pending || 0 }} · 失败 {{ robot.queue?.failed || 0 }} ·
            {{ checkedAt(robot.last_checked_at) }}
          </div>
        </header>

        <div v-if="robot.error_code" class="bot-robot-error">
          {{ robot.error_code === 'authentication_failed' ? '认证失败' : robot.error_code === 'chat_list_failed' ? '群聊读取失败' : '队列状态读取失败' }}
        </div>
        <div v-if="!robot.chats?.length" class="empty">机器人尚未加入群聊</div>
        <div v-for="chat in robot.chats" v-else :key="chat.chat_id" class="bot-chat-row">
          <div class="bot-chat-main">
            <span class="bot-chat-icon"><Bot :size="18" /></span>
            <div>
              <strong>{{ chat.title }}</strong>
              <div class="muted">
                {{ chat.joined ? (chat.chat_type === 3 ? '超级群' : '普通群') : '已不在群聊' }} · {{ chat.chat_id }}
              </div>
            </div>
          </div>
          <label class="admin-field bot-group-binding">
            <span class="admin-field-label">对应学习小组</span>
            <select
              :value="chat.group_id || 0"
              :disabled="savingBinding === `${robot.id}:${chat.chat_id}`"
              @change="assign(robot, chat, $event)"
            >
              <option :value="0">未绑定</option>
              <option v-for="group in studyGroups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </label>
        </div>
      </section>
    </div>
  </section>
</template>
