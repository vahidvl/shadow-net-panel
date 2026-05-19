<script setup>
import { onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { 
  GlobalOutlined, 
  CheckCircleFilled, 
  CloseCircleFilled, 
  LoadingOutlined,
  DisconnectOutlined,
  PlayCircleOutlined,
  UnorderedListOutlined
} from '@ant-design/icons-vue';
import { HttpUtil } from '@/utils';
import { message } from 'ant-design-vue';
import { theme } from '@/composables/useTheme.js';

const { t } = useI18n();

const status = ref({
  running: false,
  status: 'Stopped',
  configName: '',
  error: ''
});

const settings = ref({
  snPanelProxyURL: '',
  snPanelProxyEnable: false,
  snPanelProxyHistory: '[]'
});

const loading = ref(false);
const historyList = ref([]);

let timer = null;

async function fetchStatus() {
  const msg = await HttpUtil.get('/panel/setting/proxyStatus');
  if (msg?.success && msg.obj) {
    status.value = {
      running: msg.obj.running || false,
      status: msg.obj.status || 'Stopped',
      // only expose configName when truly running
      configName: msg.obj.running ? (msg.obj.configName || '') : '',
      error: msg.obj.error || ''
    };
  }
}

async function fetchSettings() {
  const msg = await HttpUtil.post('/panel/setting/all');
  if (msg?.success && msg.obj) {
    settings.value.snPanelProxyURL = msg.obj.snPanelProxyURL;
    settings.value.snPanelProxyEnable = msg.obj.snPanelProxyEnable;
    settings.value.snPanelProxyHistory = msg.obj.snPanelProxyHistory;
    
    try {
      historyList.value = JSON.parse(msg.obj.snPanelProxyHistory || '[]');
    } catch (e) {
      historyList.value = [];
    }
  }
}

async function disableProxy() {
  loading.value = true;
  try {
    const msg = await HttpUtil.post('/panel/api/inbounds/disableProxy');
    if (msg?.success) {
      message.success('Proxy disabled successfully!');
      status.value.running = false;
      status.value.status = 'Stopped';
      settings.value.snPanelProxyEnable = false;
      await fetchStatus();
      await fetchSettings();
    } else {
      message.error(msg?.msg || 'Failed to disable proxy');
    }
  } catch (err) {
    message.error(err.message || 'Error disabling proxy');
  } finally {
    loading.value = false;
  }
}

async function quickActivateProxy(proxy) {
  loading.value = true;
  try {
    const testMsg = await HttpUtil.post('/panel/setting/testProxy', { proxyUrl: proxy.url });
    if (!testMsg?.success) {
      message.error('Proxy connectivity test failed! Cannot activate.');
      return;
    }

    const updatedSettings = {
      ...settings.value,
      snPanelProxyURL: proxy.url,
      snPanelProxyEnable: true
    };

    const saveMsg = await HttpUtil.post('/panel/setting/update', updatedSettings);
    if (saveMsg?.success) {
      message.success(`Proxy "${proxy.remark}" activated successfully!`);
      await fetchStatus();
      await fetchSettings();
    } else {
      message.error('Failed to update active proxy settings.');
    }
  } catch (err) {
    message.error(err.message || 'Error activating proxy');
  } finally {
    loading.value = false;
  }
}

function handleVisibleChange(visible) {
  if (visible) {
    fetchStatus();
    fetchSettings();
  }
}

onMounted(() => {
  fetchStatus();
  fetchSettings();
  timer = setInterval(() => {
    fetchStatus();
  }, 5000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});
</script>

<template>
  <div class="proxy-status-container" :class="{ 'is-active': status.running }">
    <a-popover 
      placement="bottomRight" 
      trigger="click" 
      :overlayClassName="theme.isDark ? 'is-dark-popover' : ''"
      @visibleChange="handleVisibleChange"
    >
      <template #title>
        <div class="popover-title">
          <GlobalOutlined class="title-icon" /> Shadow-Net Panel Proxy
        </div>
      </template>
      <template #content>
        <div class="proxy-popover-content">
          <!-- Status Banner -->
          <div class="status-banner" :class="{ 'active': status.running }">
            <span class="status-label">Status:</span>
            <span class="status-value">
              <span class="pulse-dot" v-if="status.running"></span>
              {{ status.running ? 'Running' : status.status }}
            </span>
          </div>

          <!-- Active Configuration Info -->
          <div class="active-config-info" v-if="settings.snPanelProxyURL">
            <div class="info-title">Active Outbound Link:</div>
            <div class="info-remark" v-if="status.configName">
              <b>Remark:</b> {{ status.configName }}
            </div>
            <div class="info-url">{{ settings.snPanelProxyURL }}</div>
          </div>

          <div class="error-info" v-if="status.error">
            <div class="info-title" style="color: #ff4d4f;">Error:</div>
            <div class="error-text">{{ status.error }}</div>
          </div>

          <!-- Toggle / Disable Control -->
          <div class="control-actions">
            <a-button 
              v-if="status.running" 
              type="primary" 
              danger 
              block 
              :loading="loading"
              @click="disableProxy"
            >
              <template #icon><DisconnectOutlined /></template>
              Disable Proxy Outbound
            </a-button>
            <a-button 
              v-else-if="settings.snPanelProxyURL"
              type="primary" 
              block 
              :loading="loading"
              @click="quickActivateProxy({ remark: status.configName || 'Active Config', url: settings.snPanelProxyURL })"
            >
              <template #icon><PlayCircleOutlined /></template>
              Enable Panel Proxy
            </a-button>
          </div>

          <!-- Saved configurations list inside popover -->
          <div class="popover-saved-list" v-if="historyList.length > 0">
            <div class="list-header"><UnorderedListOutlined /> Quick Select Relay</div>
            <div class="quick-items-wrapper">
              <div 
                v-for="item in historyList" 
                :key="item.id" 
                class="quick-select-item"
                :class="{ 'active': status.running && settings.snPanelProxyURL === item.url }"
                @click="quickActivateProxy(item)"
              >
                <div class="item-header">
                  <span class="item-remark">{{ item.remark }}</span>
                  <span v-if="status.running && settings.snPanelProxyURL === item.url" class="active-dot"></span>
                </div>
                <div class="item-url">{{ item.url }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Badge trigger -->
      <div class="status-badge">
        <GlobalOutlined class="icon-global" />
        <span class="config-name">{{ status.running ? (status.configName || 'Active') : 'Proxy Off' }}</span>
        <div class="indicator">
          <CheckCircleFilled v-if="status.running && !loading" class="status-icon success" />
          <CloseCircleFilled v-else-if="status.status === 'Error' && !loading" class="status-icon error" />
          <LoadingOutlined v-else-if="status.status === 'Starting' || loading" class="status-icon warning" />
          <div v-if="status.running && !loading" class="pulse"></div>
        </div>
      </div>
    </a-popover>
  </div>
</template>

<style scoped>
.proxy-status-container {
  display: flex;
  align-items: center;
  padding: 5px 14px;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 20px;
  margin: 0 8px;
  transition: all 0.3s ease;
  cursor: pointer;
  border: 1px solid rgba(0, 0, 0, 0.08);
}

.proxy-status-container:hover {
  background: rgba(0, 0, 0, 0.08);
  border-color: rgba(0, 0, 0, 0.15);
  box-shadow: 0 2px 8px rgba(0,0,0,0.05);
}

.is-active {
  background: rgba(82, 196, 26, 0.08) !important;
  border-color: rgba(82, 196, 26, 0.25) !important;
}

.is-active:hover {
  background: rgba(82, 196, 26, 0.12) !important;
  border-color: rgba(82, 196, 26, 0.4) !important;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  position: relative;
}

.icon-global {
  font-size: 16px;
  color: #8c8c8c;
}

.is-active .icon-global {
  color: #52c41a;
  animation: rotate 10s linear infinite;
}

.config-name {
  font-size: 12px;
  font-weight: 600;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #595959;
}

.indicator {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
}

.status-icon {
  font-size: 12px;
  z-index: 2;
}

.success { color: #52c41a; }
.error { color: #ff4d4f; }
.warning { color: #faad14; }

.pulse {
  position: absolute;
  width: 100%;
  height: 100%;
  background: #52c41a;
  border-radius: 50%;
  opacity: 0.6;
  animation: pulse-ring 1.5s cubic-bezier(0.455, 0.03, 0.515, 0.955) infinite;
  z-index: 1;
}

/* Popover Content Styling */
.popover-title {
  font-weight: 700;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-icon {
  color: #1890ff;
}

.proxy-popover-content {
  width: 280px;
  padding: 4px 0;
}

.status-banner {
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  padding: 8px 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  border: 1px solid #f0f0f0;
}

.status-banner.active {
  background: rgba(82, 196, 26, 0.04);
  border-color: rgba(82, 196, 26, 0.2);
}

.status-label {
  font-size: 12px;
  color: #8c8c8c;
}

.status-value {
  font-weight: 700;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #8c8c8c;
}

.status-banner.active .status-value {
  color: #52c41a;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  background: #52c41a;
  border-radius: 50%;
  display: inline-block;
  animation: pulse-dot-anim 1.5s infinite;
}

.active-config-info {
  background: rgba(24, 144, 255, 0.02);
  border: 1px solid rgba(24, 144, 255, 0.1);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 12px;
}

.info-title {
  font-size: 11px;
  color: #8c8c8c;
  margin-bottom: 4px;
  font-weight: 600;
}

.info-remark {
  font-size: 12px;
  color: #262626;
  margin-bottom: 2px;
}

.info-url {
  font-family: monospace;
  font-size: 10.5px;
  color: #595959;
  word-break: break-all;
  max-height: 48px;
  overflow-y: auto;
}

.error-info {
  background: rgba(255, 77, 79, 0.02);
  border: 1px solid rgba(255, 77, 79, 0.1);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 12px;
}

.error-text {
  font-size: 11px;
  color: #ff4d4f;
  word-break: break-all;
}

.control-actions {
  margin-bottom: 16px;
}

.popover-saved-list {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.list-header {
  font-size: 12px;
  font-weight: 600;
  color: #8c8c8c;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-items-wrapper {
  max-height: 150px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.quick-select-item {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-select-item:hover {
  background: #f0f2f5;
  border-color: #d9d9d9;
}

.quick-select-item.active {
  background: rgba(24, 144, 255, 0.02);
  border-color: #1890ff;
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2px;
}

.item-remark {
  font-weight: 600;
  font-size: 11.5px;
  color: #262626;
}

.active-dot {
  width: 6px;
  height: 6px;
  background: #1890ff;
  border-radius: 50%;
}

.item-url {
  font-family: monospace;
  font-size: 9.5px;
  color: #8c8c8c;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes pulse-ring {
  0% { transform: scale(0.8); opacity: 0.5; }
  100% { transform: scale(2.4); opacity: 0; }
}

@keyframes pulse-dot-anim {
  0% { transform: scale(0.9); opacity: 0.8; }
  50% { transform: scale(1.1); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.8; }
}

/* Light/Dark mode support */
:global(body.dark) .proxy-status-container {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.12);
}

:global(body.dark) .proxy-status-container:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.2);
}

:global(body.dark) .config-name {
  color: rgba(255, 255, 255, 0.65);
}

:global(body.dark) .is-active .config-name {
  color: #52c41a;
}

:global(body.dark) .is-active {
  background: rgba(82, 196, 26, 0.12) !important;
  border-color: rgba(82, 196, 26, 0.3) !important;
}

:global(html[data-theme='ultra-dark']) .proxy-status-container {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.15);
}

:global(html[data-theme='ultra-dark']) .proxy-status-container:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.25);
}

:global(html[data-theme='ultra-dark']) .config-name {
  color: rgba(255, 255, 255, 0.7);
}

:global(html[data-theme='ultra-dark']) .is-active .config-name {
  color: #52c41a;
}

:global(html[data-theme='ultra-dark']) .is-active {
  background: rgba(82, 196, 26, 0.15) !important;
  border-color: rgba(82, 196, 26, 0.35) !important;
}

/* Dark mode popover override */
:global(.is-dark-popover) :global(.ant-popover-inner),
:global(.is-dark-popover) :global(.ant-popover-title) {
  background-color: #1f1f1f !important;
  color: rgba(255, 255, 255, 0.85) !important;
  border-color: #303030 !important;
}

:global(.is-dark-popover) .status-banner {
  background: rgba(255, 255, 255, 0.02);
  border-color: #303030;
}

:global(.is-dark-popover) .status-banner.active {
  background: rgba(82, 196, 26, 0.08);
  border-color: rgba(82, 196, 26, 0.3);
}

:global(.is-dark-popover) .active-config-info {
  background: rgba(24, 144, 255, 0.04);
  border-color: rgba(24, 144, 255, 0.2);
}

:global(.is-dark-popover) .info-remark {
  color: #e8e8e8;
}

:global(.is-dark-popover) .info-url {
  color: #a6a6a6;
}

:global(.is-dark-popover) .popover-saved-list {
  border-color: #303030;
}

:global(.is-dark-popover) .quick-select-item {
  background: #141414;
  border-color: #303030;
}

:global(.is-dark-popover) .quick-select-item:hover {
  background: #262626;
  border-color: #434343;
}

:global(.is-dark-popover) .quick-select-item.active {
  background: rgba(24, 144, 255, 0.04);
  border-color: #1890ff;
}

:global(.is-dark-popover) .item-remark {
  color: #e8e8e8;
}
</style>
