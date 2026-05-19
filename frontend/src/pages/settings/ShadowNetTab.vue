<script setup>
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { 
  ThunderboltOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  GlobalOutlined,
  CloseCircleOutlined
} from '@ant-design/icons-vue';
import SettingListItem from '@/components/SettingListItem.vue';
import { HttpUtil } from '@/utils';
import { message } from 'ant-design-vue';

const { t } = useI18n();

const props = defineProps({
  allSetting: { type: Object, required: true },
});

const testingProxy = ref(false);
const proxyTestResult = ref(null);

async function testProxyConnection(urlStr) {
  testingProxy.value = true;
  proxyTestResult.value = null;
  try {
    const msg = await HttpUtil.post('/panel/setting/testProxy', { proxyUrl: urlStr });
    proxyTestResult.value = {
      success: msg.success,
      message: msg.msg || (msg.success ? 'Connection Successful!' : 'Connection Failed')
    };
    return msg.success;
  } catch (err) {
    proxyTestResult.value = {
      success: false,
      message: err.message || 'Network Error'
    };
    return false;
  } finally {
    testingProxy.value = false;
  }
}

// History CRUD logic
const historyList = computed(() => {
  try {
    const arr = JSON.parse(props.allSetting.snPanelProxyHistory || '[]');
    return Array.isArray(arr) ? arr : [];
  } catch (e) {
    return [];
  }
});

function updateHistory(newHistory) {
  props.allSetting.snPanelProxyHistory = JSON.stringify(newHistory);
}

const modalVisible = ref(false);
const modalMode = ref('add'); // 'add' or 'edit'
const editIndex = ref(-1);
const savingProxy = ref(false);
const formState = ref({
  remark: '',
  url: ''
});

function showAddModal() {
  modalMode.value = 'add';
  formState.value = { remark: '', url: '' };
  modalVisible.value = true;
}

function showEditModal(index, item) {
  modalMode.value = 'edit';
  editIndex.value = index;
  formState.value = { remark: item.remark, url: item.url };
  modalVisible.value = true;
}

async function handleModalOk() {
  if (!formState.value.remark.trim()) {
    message.error('Please enter a remark name');
    return;
  }
  if (!formState.value.url.trim()) {
    message.error('Please enter a valid proxy URL');
    return;
  }

  savingProxy.value = true;
  const isOk = await testProxyConnection(formState.value.url);
  savingProxy.value = false;
  
  if (!isOk) {
    message.error('Proxy connection test failed! Cannot save this configuration.');
    return;
  }

  const currentList = [...historyList.value];
  if (modalMode.value === 'add') {
    currentList.push({
      id: String(Date.now()),
      remark: formState.value.remark,
      url: formState.value.url,
      addedAt: new Date().toISOString()
    });
  } else {
    currentList[editIndex.value] = {
      ...currentList[editIndex.value],
      remark: formState.value.remark,
      url: formState.value.url
    };
  }

  updateHistory(currentList);
  modalVisible.value = false;
  message.success('Proxy configuration saved to history successfully!');
}

const activatingProxyId = ref(null);

async function handleProxyToggle(checked) {
  if (checked) {
    if (!props.allSetting.snPanelProxyURL || !props.allSetting.snPanelProxyURL.trim()) {
      message.error('Please enter a proxy URL or activate one from history first.');
      return;
    }
    const isOk = await testProxyConnection(props.allSetting.snPanelProxyURL);
    if (isOk) {
      props.allSetting.snPanelProxyEnable = true;
      message.success('Proxy connection test passed! Click Save at the top to apply.');
    } else {
      message.error('Proxy connection test failed! Cannot enable.');
    }
  } else {
    props.allSetting.snPanelProxyEnable = false;
  }
}

async function activateProxy(item) {
  activatingProxyId.value = item.id;
  try {
    const isOk = await testProxyConnection(item.url);
    if (!isOk) {
      message.error('Proxy connection test failed! Cannot activate this proxy.');
      return;
    }

    props.allSetting.snPanelProxyURL = item.url;
    props.allSetting.snPanelProxyEnable = true;
    message.success(`Proxy "${item.remark}" activated! Click Save at the top to apply.`);
  } catch (err) {
    message.error('Error activating proxy: ' + err.message);
  } finally {
    activatingProxyId.value = null;
  }
}

function deleteProxy(index) {
  const currentList = [...historyList.value];
  const item = currentList[index];
  currentList.splice(index, 1);
  updateHistory(currentList);
  message.success(`Proxy "${item.remark}" removed from history.`);
}
</script>

<template>
  <a-collapse default-active-key="1">
    <a-collapse-panel key="1" header="Shadow-Net Bot Configuration">
      <SettingListItem paddings="small">
        <template #title>Sales Bot Token</template>
        <template #description>
          {{ allSetting.hasSnBotTokenSales ? 'Configured; leave blank to keep current token.' : 'The Telegram bot token for the Sales agent.' }}
        </template>
        <template #control>
          <a-input-password v-model:value="allSetting.snBotTokenSales"
            :placeholder="allSetting.hasSnBotTokenSales ? 'Configured - enter a new token to replace' : ''" />
        </template>
      </SettingListItem>

      <SettingListItem paddings="small">
        <template #title>Sentinel Bot Token</template>
        <template #description>
          {{ allSetting.hasSnBotTokenSentinel ? 'Configured; leave blank to keep current token.' : 'The Telegram bot token for the Sentinel (IP Monitoring) agent.' }}
        </template>
        <template #control>
          <a-input-password v-model:value="allSetting.snBotTokenSentinel"
            :placeholder="allSetting.hasSnBotTokenSentinel ? 'Configured - enter a new token to replace' : ''" />
        </template>
      </SettingListItem>

      <SettingListItem paddings="small">
        <template #title>Admin Bot Token</template>
        <template #description>
          {{ allSetting.hasSnBotTokenAdmin ? 'Configured; leave blank to keep current token.' : 'The Telegram bot token for the Administrator agent.' }}
        </template>
        <template #control>
          <a-input-password v-model:value="allSetting.snBotTokenAdmin"
            :placeholder="allSetting.hasSnBotTokenAdmin ? 'Configured - enter a new token to replace' : ''" />
        </template>
      </SettingListItem>

      <SettingListItem paddings="small">
        <template #title>Admin Chat ID</template>
        <template #description>Telegram Chat ID for receiving administrative alerts.</template>
        <template #control>
          <a-input v-model:value="allSetting.snAdminChatId" type="text" placeholder="e.g. 12345678" />
        </template>
      </SettingListItem>
    </a-collapse-panel>

    <a-collapse-panel key="2" header="Penalty System (3-Strikes)">
      <SettingListItem paddings="small">
        <template #title>Max Penalty Strikes</template>
        <template #description>Number of IP limit violations allowed before automatically disabling the account.</template>
        <template #control>
          <a-input-number v-model:value="allSetting.snMaxPenalty" :min="1" :max="10" :style="{ width: '100%' }" />
        </template>
      </SettingListItem>
    </a-collapse-panel>
    
    <a-collapse-panel key="3" header="Panel Outbound Proxy (Global Connectivity)">
      <div class="proxy-section-container">
        <SettingListItem paddings="small">
          <template #title>Enable Panel Proxy</template>
          <template #description>
            Route all panel outgoing requests (GitHub updates, IP detection, etc.) through a proxy. 
            <br/>
            <span style="color: #1890ff">Essential for servers with restricted international access.</span>
          </template>
          <template #control>
            <a-switch :checked="allSetting.snPanelProxyEnable" :loading="testingProxy" @change="handleProxyToggle" />
          </template>
        </SettingListItem>

        <transition name="fade">
          <div v-if="allSetting.snPanelProxyEnable" class="proxy-details">
            <SettingListItem paddings="small">
              <template #title>Active Proxy URL / Link</template>
              <template #description>
                Support standard proxies (<code>http</code>, <code>socks5</code>) and direct protocol links (<code>vless://</code>, <code>vmess://</code>, <code>trojan://</code>).
              </template>
              <template #control>
                <a-input v-model:value="allSetting.snPanelProxyURL" placeholder="e.g. vless://uuid@host:port?security=reality..." />
              </template>
            </SettingListItem>

            <div class="test-proxy-wrapper">
              <a-button type="dashed" :loading="testingProxy" @click="testProxyConnection(allSetting.snPanelProxyURL)" block>
                <template #icon><ThunderboltOutlined v-if="!testingProxy"/></template>
                Test Connectivity to GitHub
              </a-button>
              
              <transition name="slide-up">
                <a-alert v-if="proxyTestResult" 
                         :type="proxyTestResult.success ? 'success' : 'error'" 
                         :message="proxyTestResult.message" 
                         show-icon 
                         class="test-alert" />
              </transition>
            </div>
          </div>
        </transition>

        <!-- Saved Configurations Section -->
        <div class="saved-configs-section">
          <div class="saved-configs-header">
            <span class="section-title"><GlobalOutlined /> Saved Proxy Configurations</span>
            <a-button type="primary" size="small" @click="showAddModal">
              <template #icon><PlusOutlined /></template> Add Proxy
            </a-button>
          </div>

          <div v-if="historyList.length === 0" class="empty-history">
            No saved configurations in history yet. Inbound proxies or custom links will appear here.
          </div>

          <div v-else class="history-list">
            <div 
              v-for="(item, index) in historyList" 
              :key="item.id || index" 
              class="proxy-card"
              :class="{ 'active': allSetting.snPanelProxyEnable && allSetting.snPanelProxyURL === item.url }"
            >
              <div class="proxy-card-header">
                <span class="proxy-remark">{{ item.remark }}</span>
                <span v-if="allSetting.snPanelProxyEnable && allSetting.snPanelProxyURL === item.url" class="active-badge">
                  <span class="pulse-dot"></span> Active Proxy
                </span>
              </div>
              <div class="proxy-card-body">
                <span class="proxy-url-label">URL/Config:</span>
                <div class="proxy-url-text">{{ item.url }}</div>
              </div>
              <div class="proxy-card-actions">
                <a-button 
                  type="primary" 
                  size="small" 
                  :disabled="allSetting.snPanelProxyEnable && allSetting.snPanelProxyURL === item.url"
                  :loading="activatingProxyId === item.id"
                  @click="activateProxy(item)"
                >
                  <template #icon><PlayCircleOutlined /></template> Activate
                </a-button>
                <a-space>
                  <a-button size="small" @click="showEditModal(index, item)">
                    <template #icon><EditOutlined /></template> Edit
                  </a-button>
                  <a-popconfirm
                    title="Are you sure you want to delete this configuration?"
                    ok-text="Yes"
                    cancel-text="No"
                    @confirm="deleteProxy(index)"
                  >
                    <a-button type="primary" danger size="small">
                      <template #icon><DeleteOutlined /></template> Delete
                    </a-button>
                  </a-popconfirm>
                </a-space>
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-collapse-panel>
  </a-collapse>

  <!-- Add/Edit Modal -->
  <a-modal
    v-model:visible="modalVisible"
    :title="modalMode === 'add' ? 'Add Proxy Configuration' : 'Edit Proxy Configuration'"
    @ok="handleModalOk"
    :confirm-loading="savingProxy"
    ok-text="Test & Save"
  >
    <a-form layout="vertical">
      <a-form-item label="Remark / Name">
        <a-input v-model:value="formState.remark" placeholder="e.g. Tehran Relay" />
      </a-form-item>
      <a-form-item label="Proxy URL / Link">
        <a-textarea 
          v-model:value="formState.url" 
          placeholder="e.g. vless://uuid@host:port?security=reality..." 
          :auto-size="{ minRows: 2, maxRows: 6 }"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped>
.proxy-section-container {
  padding: 8px;
}
.proxy-details {
  margin-top: 16px;
  padding: 16px;
  background: rgba(24, 144, 255, 0.05);
  border-radius: 12px;
  border: 1px dashed rgba(24, 144, 255, 0.2);
}
.test-proxy-wrapper {
  margin-top: 16px;
}
.test-alert {
  margin-top: 12px;
}
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
.slide-up-enter-active, .slide-up-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-up-enter-from {
  transform: translateY(10px);
  opacity: 0;
}

/* History Styles */
.saved-configs-section {
  margin-top: 32px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  padding-top: 24px;
}
.saved-configs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-title {
  font-size: 16px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.empty-history {
  padding: 24px;
  text-align: center;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  color: #8c8c8c;
  font-size: 13px;
  border: 1px dashed rgba(0, 0, 0, 0.06);
}
.history-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.proxy-card {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  padding: 16px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.01);
}
.proxy-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
}
.proxy-card.active {
  background: rgba(24, 144, 255, 0.02);
  border-color: #1890ff;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.06);
}
.proxy-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.proxy-remark {
  font-weight: 600;
  font-size: 15px;
}
.active-badge {
  background: #e6f7ff;
  color: #1890ff;
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid #91d5ff;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.pulse-dot {
  width: 6px;
  height: 6px;
  background-color: #1890ff;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}
.proxy-card-body {
  background: rgba(0, 0, 0, 0.01);
  padding: 8px 12px;
  border-radius: 8px;
  margin-bottom: 12px;
}
.proxy-url-label {
  font-size: 11px;
  color: #8c8c8c;
  display: block;
  margin-bottom: 4px;
}
.proxy-url-text {
  font-family: monospace;
  font-size: 12px;
  color: #595959;
  word-break: break-all;
  max-height: 48px;
  overflow-y: auto;
}
.proxy-card-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

@keyframes pulse {
  0% {
    transform: scale(0.9);
    box-shadow: 0 0 0 0 rgba(24, 144, 255, 0.7);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 6px rgba(24, 144, 255, 0);
  }
  100% {
    transform: scale(0.9);
    box-shadow: 0 0 0 0 rgba(24, 144, 255, 0);
  }
}
</style>
