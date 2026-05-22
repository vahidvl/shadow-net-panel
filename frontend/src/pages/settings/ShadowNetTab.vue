<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { 
  ThunderboltOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  GlobalOutlined,
  CloseCircleOutlined,
  SettingOutlined,
  WalletOutlined,
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  InfoCircleOutlined
} from '@ant-design/icons-vue';
import SettingListItem from '@/components/SettingListItem.vue';
import { HttpUtil } from '@/utils';
import { message } from 'ant-design-vue';

const { t } = useI18n();

const props = defineProps({
  allSetting: { type: Object, required: true },
});

const basePath = window.X_UI_BASE_PATH || '';

// Proxy state
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

// Bot Plans State & CRUD
const plans = ref([]);
const loadingPlans = ref(false);
const planModalVisible = ref(false);
const planModalMode = ref('add'); // 'add' or 'edit'
const planFormState = ref({
  id: 0,
  name: '',
  volume_gb: 50,
  days: 30,
  price_toman: 50000,
  enabled: 1
});

async function fetchPlans() {
  loadingPlans.value = true;
  try {
    const res = await HttpUtil.get(basePath + 'api/bot/config');
    if (res?.status === 'success' || res?.plans) {
      plans.value = res.plans || [];
    } else if (res?.success) {
      // In case HttpUtil wrapped it in Msg
      plans.value = res.obj?.plans || [];
    }
  } catch (err) {
    message.error('Failed to load plans');
  } finally {
    loadingPlans.value = false;
  }
}

function showAddPlanModal() {
  planModalMode.value = 'add';
  planFormState.value = {
    id: 0,
    name: '',
    volume_gb: 50,
    days: 30,
    price_toman: 50000,
    enabled: 1
  };
  planModalVisible.value = true;
}

function showEditPlanModal(plan) {
  planModalMode.value = 'edit';
  planFormState.value = {
    id: plan.id,
    name: plan.name,
    volume_gb: plan.volume_gb,
    days: plan.days,
    price_toman: plan.price_toman,
    enabled: plan.enabled === undefined ? 1 : plan.enabled
  };
  planModalVisible.value = true;
}

async function handlePlanModalOk() {
  if (!planFormState.value.name.trim()) {
    message.error('Please enter package name');
    return;
  }

  const payload = {
    ...planFormState.value,
    action: planModalMode.value
  };

  try {
    const res = await HttpUtil.post(basePath + 'api/bot/plans/manage', payload);
    if (res?.status === 'success' || res?.success) {
      message.success(`Package successfully ${planModalMode.value === 'add' ? 'created' : 'updated'}!`);
      planModalVisible.value = false;
      fetchPlans();
    } else {
      message.error(res?.msg || 'Failed to save plan');
    }
  } catch (err) {
    message.error('Error saving plan: ' + err.message);
  }
}

async function deletePlan(planId) {
  try {
    const res = await HttpUtil.post(basePath + 'api/bot/plans/manage', {
      id: planId,
      action: 'delete'
    });
    if (res?.status === 'success' || res?.success) {
      message.success('Package deleted successfully');
      fetchPlans();
    } else {
      message.error(res?.msg || 'Failed to delete plan');
    }
  } catch (err) {
    message.error('Error deleting plan: ' + err.message);
  }
}

async function togglePlanStatus(plan) {
  try {
    const res = await HttpUtil.post(basePath + 'api/bot/plans/manage', {
      id: plan.id,
      action: 'toggle'
    });
    if (res?.status === 'success' || res?.success) {
      message.success('Package status toggled successfully');
      fetchPlans();
    } else {
      message.error(res?.msg || 'Failed to toggle status');
    }
  } catch (err) {
    message.error('Error toggling status: ' + err.message);
  }
}

// Pending Transactions State
const pendingTxs = ref([]);
const loadingTxs = ref(false);

async function fetchPendingTxs() {
  loadingTxs.value = true;
  try {
    const res = await HttpUtil.get(basePath + 'api/transactions/pending');
    // HttpUtil might return Msg or raw array
    if (Array.isArray(res)) {
      pendingTxs.value = res;
    } else if (res?.obj && Array.isArray(res.obj)) {
      pendingTxs.value = res.obj;
    } else if (res?.data && Array.isArray(res.data)) {
      pendingTxs.value = res.data;
    } else {
      pendingTxs.value = [];
    }
  } catch (err) {
    message.error('Failed to load pending transactions');
  } finally {
    loadingTxs.value = false;
  }
}

async function processTransaction(txId, action) {
  try {
    const res = await HttpUtil.post(basePath + 'api/transactions/action', {
      transaction_id: txId,
      action: action
    });
    if (res?.status === 'success' || res?.success) {
      message.success(`Transaction ${action === 'approve' ? 'approved' : 'rejected'} successfully!`);
      fetchPendingTxs();
    } else {
      message.error(res?.msg || 'Failed to process transaction');
    }
  } catch (err) {
    message.error('Error processing transaction: ' + err.message);
  }
}

onMounted(() => {
  fetchPlans();
  fetchPendingTxs();
});
</script>

<template>
  <a-collapse default-active-key="1">
    <!-- Collapse 1: Bot APIs & Tokens -->
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

    <!-- Collapse 2: General & Direct Support -->
    <a-collapse-panel key="2" header="Bot Features & Support Settings">
      <SettingListItem paddings="small">
        <template #title>Direct Support Button</template>
        <template #description>Show a direct support contact button to users inside the bot interface.</template>
        <template #control>
          <a-switch v-model:checked="allSetting.snBotSupportEnabled" />
        </template>
      </SettingListItem>

      <SettingListItem paddings="small">
        <template #title>Max Penalty Strikes</template>
        <template #description>Number of IP limit violations allowed before automatically disabling the account.</template>
        <template #control>
          <a-input-number v-model:value="allSetting.snMaxPenalty" :min="1" :max="10" :style="{ width: '100%' }" />
        </template>
      </SettingListItem>
    </a-collapse-panel>

    <!-- Collapse 3: Free Trial System Configuration -->
    <a-collapse-panel key="3" header="Free Trial Package Configuration">
      <SettingListItem paddings="small">
        <template #title>Enable Free Trial</template>
        <template #description>Allow new users to request a free trial subscription.</template>
        <template #control>
          <a-switch v-model:checked="allSetting.snBotEnableTrial" />
        </template>
      </SettingListItem>

      <transition name="fade">
        <div v-if="allSetting.snBotEnableTrial" class="trial-details-panel">
          <SettingListItem paddings="small">
            <template #title>Trial Volume (MB)</template>
            <template #description>Data limit allocated for trial subscriptions.</template>
            <template #control>
              <a-input-number v-model:value="allSetting.snBotTrialVolumeMb" :min="100" :style="{ width: '100%' }" />
            </template>
          </SettingListItem>

          <SettingListItem paddings="small">
            <template #title>Trial Duration (Hours)</template>
            <template #description>Validity duration of the trial before expiration.</template>
            <template #control>
              <a-input-number v-model:value="allSetting.snBotTrialExpiryHours" :min="1" :style="{ width: '100%' }" />
            </template>
          </SettingListItem>

          <SettingListItem paddings="small">
            <template #title>Trial IP Limit</template>
            <template #description>Maximum number of concurrent IPs allowed to use the trial subscription.</template>
            <template #control>
              <a-input-number v-model:value="allSetting.snBotTrialIpLimit" :min="1" :max="5" :style="{ width: '100%' }" />
            </template>
          </SettingListItem>
        </div>
      </transition>
    </a-collapse-panel>

    <!-- Collapse 4: Payments Configuration -->
    <a-collapse-panel key="4" header="Payment Methods & Gateways">
      <SettingListItem paddings="small">
        <template #title>Enable Online Payment Gateway (Zibal)</template>
        <template #description>Allow users to pay online using the Zibal gateway.</template>
        <template #control>
          <a-switch v-model:checked="allSetting.snBotEnableZibal" />
        </template>
      </SettingListItem>

      <SettingListItem paddings="small">
        <template #title>Enable Card-to-Card Payment</template>
        <template #description>Allow users to buy packages via manual card transfer and upload bank receipt.</template>
        <template #control>
          <a-switch v-model:checked="allSetting.snBotEnableCardToCard" />
        </template>
      </SettingListItem>

      <transition name="fade">
        <div v-if="allSetting.snBotEnableCardToCard" class="card-details-panel">
          <SettingListItem paddings="small">
            <template #title>Bank Card Number</template>
            <template #description>16-digit card number displayed to users for transfer.</template>
            <template #control>
              <a-input v-model:value="allSetting.snBotCardNumber" placeholder="e.g. 6037997912345678" />
            </template>
          </SettingListItem>

          <SettingListItem paddings="small">
            <template #title>Cardholder Full Name</template>
            <template #description>Account owner name displayed for card validation.</template>
            <template #control>
              <a-input v-model:value="allSetting.snBotCardOwner" placeholder="e.g. Vahid ..." />
            </template>
          </SettingListItem>
        </div>
      </transition>
    </a-collapse-panel>

    <!-- Collapse 5: Bot Plans/Packages Management -->
    <a-collapse-panel key="5" header="Sales Packages & Plans">
      <div class="plans-section-container">
        <div class="section-action-header">
          <span class="custom-sec-title"><WalletOutlined /> Subscription Packages</span>
          <a-space>
            <a-button type="default" size="small" :loading="loadingPlans" @click="fetchPlans">
              <template #icon><ReloadOutlined /></template> Refresh
            </a-button>
            <a-button type="primary" size="small" @click="showAddPlanModal">
              <template #icon><PlusOutlined /></template> Create Package
            </a-button>
          </a-space>
        </div>

        <div v-if="plans.length === 0" class="empty-list-placeholder">
          No subscription packages configured. Create one to display packages in the sales bot.
        </div>

        <div v-else class="plans-grid">
          <div 
            v-for="plan in plans" 
            :key="plan.id" 
            class="plan-item-card"
            :class="{ 'plan-disabled': plan.enabled !== 1 }"
          >
            <div class="plan-card-header">
              <span class="plan-card-name">{{ plan.name }}</span>
              <a-tag :color="plan.enabled === 1 ? 'success' : 'default'">
                {{ plan.enabled === 1 ? 'Active' : 'Inactive' }}
              </a-tag>
            </div>
            <div class="plan-card-metrics">
              <div class="metric-row">
                <span class="metric-label">Volume:</span>
                <span class="metric-value">{{ plan.volume_gb > 0 ? plan.volume_gb + ' GB' : 'Unlimited' }}</span>
              </div>
              <div class="metric-row">
                <span class="metric-label">Duration:</span>
                <span class="metric-value">{{ plan.days > 0 ? plan.days + ' Days' : 'Lifetime' }}</span>
              </div>
              <div class="metric-row">
                <span class="metric-label">Price:</span>
                <span class="metric-value highlight-price">{{ plan.price_toman.toLocaleString() }} Toman</span>
              </div>
            </div>
            <div class="plan-card-footer">
              <a-button size="small" type="default" @click="showEditPlanModal(plan)">
                <template #icon><EditOutlined /></template> Edit
              </a-button>
              <a-space>
                <a-button 
                  size="small" 
                  :type="plan.enabled === 1 ? 'default' : 'primary'"
                  @click="togglePlanStatus(plan)"
                >
                  {{ plan.enabled === 1 ? 'Deactivate' : 'Activate' }}
                </a-button>
                <a-popconfirm
                  title="Are you sure you want to delete this subscription package?"
                  ok-text="Yes"
                  cancel-text="No"
                  @confirm="deletePlan(plan.id)"
                >
                  <a-button size="small" type="primary" danger>
                    <template #icon><DeleteOutlined /></template> Delete
                  </a-button>
                </a-popconfirm>
              </a-space>
            </div>
          </div>
        </div>
      </div>
    </a-collapse-panel>

    <!-- Collapse 6: Pending bank receipt approvals -->
    <a-collapse-panel key="6" header="Pending Bank Receipts Approval">
      <div class="txs-section-container">
        <div class="section-action-header">
          <span class="custom-sec-title"><InfoCircleOutlined /> Pending Receipts Queue</span>
          <a-button type="default" size="small" :loading="loadingTxs" @click="fetchPendingTxs">
            <template #icon><ReloadOutlined /></template> Refresh Queue
          </a-button>
        </div>

        <div v-if="pendingTxs.length === 0" class="empty-list-placeholder">
          No bank receipts pending review. High-five!
        </div>

        <div v-else class="txs-list">
          <div 
            v-for="tx in pendingTxs" 
            :key="tx.id" 
            class="tx-item-card"
          >
            <div class="tx-card-meta-grid">
              <div class="tx-meta-col">
                <span class="tx-meta-label">Transaction ID</span>
                <span class="tx-meta-val">#{{ tx.id }}</span>
              </div>
              <div class="tx-meta-col">
                <span class="tx-meta-label">Telegram Username</span>
                <span class="tx-meta-val">@{{ tx.username || 'No Username' }} (<code>{{ tx.chat_id }}</code>)</span>
              </div>
              <div class="tx-meta-col">
                <span class="tx-meta-label">Requested Package</span>
                <span class="tx-meta-val">Plan ID: {{ tx.plan_id }} ({{ tx.amount.toLocaleString() }} Toman)</span>
              </div>
              <div class="tx-meta-col">
                <span class="tx-meta-label">Bank Receipt Details</span>
                <span class="tx-meta-val text-ellipsis" :title="tx.card_details">{{ tx.card_details }}</span>
              </div>
              <div class="tx-meta-col">
                <span class="tx-meta-label">Submitted At</span>
                <span class="tx-meta-val">{{ new Date(tx.created_at * 1000).toLocaleString('fa-IR') }}</span>
              </div>
            </div>
            <div class="tx-card-actions">
              <a-popconfirm
                title="Reject this payment receipt?"
                ok-text="Reject"
                cancel-text="Cancel"
                ok-button-props="{ danger: true }"
                @confirm="processTransaction(tx.id, 'reject')"
              >
                <a-button type="primary" danger size="small">
                  <template #icon><CloseOutlined /></template> Reject Receipt
                </a-button>
              </a-popconfirm>
              <a-button type="primary" size="small" class="approve-btn" @click="processTransaction(tx.id, 'approve')">
                <template #icon><CheckOutlined /></template> Approve & Issue Account
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </a-collapse-panel>

    <!-- Collapse 7: Panel Outbound Proxy -->
    <a-collapse-panel key="7" header="Panel Outbound Proxy (Global Connectivity)">
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

  <!-- Add/Edit Proxy Modal -->
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

  <!-- Add/Edit Subscription Plan Modal -->
  <a-modal
    v-model:visible="planModalVisible"
    :title="planModalMode === 'add' ? 'Create Subscription Package' : 'Edit Subscription Package'"
    @ok="handlePlanModalOk"
    ok-text="Save Package"
  >
    <a-form layout="vertical">
      <a-form-item label="Package Name (Persian/English)">
        <a-input v-model:value="planFormState.name" placeholder="e.g. ۳ ماهه ۵۰ گیگابایت" />
      </a-form-item>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="Volume Limit (GB) - 0 for Unlimited">
            <a-input-number v-model:value="planFormState.volume_gb" :min="0" :style="{ width: '100%' }" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="Duration (Days) - 0 for Lifetime">
            <a-input-number v-model:value="planFormState.days" :min="0" :style="{ width: '100%' }" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="Price (Toman)">
            <a-input-number v-model:value="planFormState.price_toman" :min="0" :style="{ width: '100%' }" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="Status">
            <a-select v-model:value="planFormState.enabled" :style="{ width: '100%' }">
              <a-select-option :value="1">Active</a-select-option>
              <a-select-option :value="0">Inactive</a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>
    </a-form>
  </a-modal>
</template>

<style scoped>
.proxy-section-container, .plans-section-container, .txs-section-container {
  padding: 8px;
}
.trial-details-panel, .card-details-panel, .proxy-details {
  margin-top: 12px;
  padding: 16px;
  background: rgba(24, 144, 255, 0.03);
  border-radius: 12px;
  border: 1px dashed rgba(24, 144, 255, 0.15);
}
.section-action-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}
.custom-sec-title {
  font-size: 15px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.empty-list-placeholder {
  padding: 32px;
  text-align: center;
  background: rgba(0, 0, 0, 0.01);
  border-radius: 10px;
  color: #8c8c8c;
  font-size: 13px;
  border: 1px dashed rgba(0, 0, 0, 0.05);
  margin-bottom: 16px;
}

/* Plans Grid CSS */
.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}
.plan-item-card {
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 12px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
  transition: all 0.3s ease;
}
.plan-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.06);
}
.plan-disabled {
  opacity: 0.65;
  background: #fafafa;
}
.plan-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.plan-card-name {
  font-size: 16px;
  font-weight: 700;
  color: #262626;
}
.plan-card-metrics {
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 16px;
}
.metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  margin-bottom: 6px;
}
.metric-row:last-child {
  margin-bottom: 0;
}
.metric-label {
  color: #8c8c8c;
}
.metric-value {
  font-weight: 600;
  color: #595959;
}
.highlight-price {
  color: #389e0a;
  font-size: 14px;
  font-weight: 700;
}
.plan-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* Pending Transactions CSS */
.txs-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 16px;
}
.tx-item-card {
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 12px;
  padding: 18px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
  transition: all 0.25s ease;
}
.tx-item-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
}
.tx-card-meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
  padding-bottom: 14px;
}
.tx-meta-col {
  display: flex;
  flex-direction: column;
}
.tx-meta-label {
  font-size: 11px;
  color: #8c8c8c;
  margin-bottom: 4px;
  text-transform: uppercase;
}
.tx-meta-val {
  font-size: 13px;
  font-weight: 600;
  color: #262626;
}
.text-ellipsis {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tx-card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
.approve-btn {
  background-color: #389e0a;
  border-color: #389e0a;
}
.approve-btn:hover, .approve-btn:focus {
  background-color: #46bc13 !important;
  border-color: #46bc13 !important;
}

/* Outbound Proxy CSS */
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

/* Proxy History CSS */
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
