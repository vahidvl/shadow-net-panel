<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { GlobalOutlined } from '@ant-design/icons-vue';
import { HttpUtil } from '@/utils';

const running = ref(false);
const configName = ref('');
let timer = null;

async function fetchStatus() {
  try {
    const msg = await HttpUtil.get('/panel/setting/proxyStatus');
    if (msg?.success && msg.obj) {
      running.value = msg.obj.running === true;
      configName.value = running.value ? (msg.obj.configName || '') : '';
    } else {
      running.value = false;
      configName.value = '';
    }
  } catch {
    running.value = false;
    configName.value = '';
  }
}

onMounted(() => {
  fetchStatus();
  timer = setInterval(fetchStatus, 5000);
});

onUnmounted(() => clearInterval(timer));
</script>

<template>
  <Transition name="proxy-badge-fade">
    <a-tooltip
      v-if="running"
      :title="configName ? `Panel Proxy Active: ${configName}` : 'Panel Proxy Active'"
      placement="bottom"
    >
      <span class="proxy-header-badge" aria-label="Proxy Active">
        <span class="pulse-ring" />
        <span class="pulse-dot" />
        <GlobalOutlined class="badge-icon" />
      </span>
    </a-tooltip>
  </Transition>
</template>

<style scoped>
.proxy-header-badge {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  cursor: default;
  vertical-align: middle;
}

/* Outer pulsing ring */
.pulse-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: rgba(82, 196, 26, 0.25);
  animation: proxy-pulse 2s ease-out infinite;
}

/* Solid center dot */
.pulse-dot {
  position: absolute;
  width: 10px;
  height: 10px;
  background: #52c41a;
  border-radius: 50%;
  box-shadow: 0 0 6px rgba(82, 196, 26, 0.7);
}

/* Icon sits on top, inherits green */
.badge-icon {
  position: relative;
  z-index: 1;
  font-size: 13px;
  color: #52c41a;
  filter: drop-shadow(0 0 3px rgba(82, 196, 26, 0.5));
}

@keyframes proxy-pulse {
  0% {
    transform: scale(0.85);
    opacity: 0.9;
  }
  70% {
    transform: scale(1.55);
    opacity: 0;
  }
  100% {
    transform: scale(0.85);
    opacity: 0;
  }
}

/* Mount/unmount fade */
.proxy-badge-fade-enter-active,
.proxy-badge-fade-leave-active {
  transition: opacity 0.4s ease, transform 0.4s ease;
}
.proxy-badge-fade-enter-from,
.proxy-badge-fade-leave-to {
  opacity: 0;
  transform: scale(0.6);
}
</style>
