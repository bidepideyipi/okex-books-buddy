<template>
  <el-card class="ws-monitor-card" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="card-title">
          <el-icon><Connection /></el-icon>
          WebSocket Connection Monitor
        </span>
        <div class="header-actions">
          <el-button 
            type="primary" 
            size="small" 
            :icon="Refresh" 
            :loading="loading"
            @click="refreshStatus"
          >
            Refresh
          </el-button>
        </div>
      </div>
    </template>

    <div v-if="error" class="error-message">
      <el-alert 
        :title="error" 
        type="error" 
        :closable="false"
        show-icon
      />
    </div>

    <div v-else-if="loading && !statusData" class="loading-container">
      <el-skeleton :rows="5" animated />
    </div>

    <div v-else-if="statusData" class="status-content">
      <!-- Summary Stats -->
      <el-row :gutter="16" class="summary-row">
        <el-col :xs="24" :sm="8">
          <el-statistic 
            title="WebSocket URL" 
            :value="websocketUrl"
          >
            <template #prefix>
              <el-icon><Link /></el-icon>
            </template>
          </el-statistic>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-statistic 
            title="Connection Status" 
          >
            <template #default>
              <el-tag 
                :type="getStatusType(connectionStatus)"
                effect="dark"
                size="large"
              >
                {{ connectionStatus }}
              </el-tag>
            </template>
          </el-statistic>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-statistic 
            title="Subscribed Pairs" 
            :value="statusData.data.active_pairs"
          >
            <template #prefix>
              <el-icon><TrendCharts /></el-icon>
            </template>
          </el-statistic>
        </el-col>
      </el-row>

      <el-divider />

      <!-- WebSocket Connection Info -->
      <el-descriptions :column="2" border class="ws-info">
        <el-descriptions-item label="Connected At">
          {{ connectedAt }}
        </el-descriptions-item>
        <el-descriptions-item label="Last Message">
          <span :class="{ 'stale-message': isMessageStale }">
            {{ lastMessageAt }}
          </span>
        </el-descriptions-item>
        <el-descriptions-item label="Reconnects">
          <el-tag 
            :type="reconnectCount > 0 ? 'warning' : 'info'" 
            size="small"
          >
            {{ reconnectCount }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Total Messages">
          {{ formatNumber(totalMessages) }}
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />
    </div>

    <!-- Details Dialog -->
    <el-dialog 
      v-model="detailsDialogVisible" 
      title="WebSocket Connection Details"
      width="600px"
    >
      <div v-if="statusData" class="connection-details">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="WebSocket URL">
            {{ websocketUrl }}
          </el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusType(connectionStatus)" effect="dark">
              {{ connectionStatus }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Connected At">
            {{ connectedAt }}
          </el-descriptions-item>
          <el-descriptions-item label="Last Message At">
            {{ lastMessageAt }}
          </el-descriptions-item>
          <el-descriptions-item label="Reconnect Count">
            {{ reconnectCount }}
          </el-descriptions-item>
          <el-descriptions-item label="Total Messages Received">
            {{ formatNumber(totalMessages) }}
          </el-descriptions-item>
          <el-descriptions-item label="Subscribed Pairs">
            {{ statusData.data.active_pairs }}
          </el-descriptions-item>
          <el-descriptions-item 
            v-if="lastError" 
            label="Last Error"
          >
            <el-text type="danger">{{ lastError }}</el-text>
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { 
  Connection, 
  Refresh, 
  Link, 
  TrendCharts, 
  CircleCheck 
} from '@element-plus/icons-vue'
import { fetchWebSocketStatus, type WSStatusResponse, type WSConnectionStatus } from '../services/api'

const props = defineProps<{
  autoRefreshProp?: boolean
}>()

const statusData = ref<WSStatusResponse | null>(null)
const loading = ref(false)
const error = ref('')
const detailsDialogVisible = ref(false)
let autoRefreshTimer: number | null = null

// WebSocket connection properties (1 connection for all pairs)
const websocketUrl = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 'N/A'
  }
  return statusData.value.data.connections[0]?.websocket_url || 'N/A'
})

const connectionStatus = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 'disconnected'
  }
  return statusData.value.data.connections[0]?.status || 'disconnected'
})

const connectedAt = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 'N/A'
  }
  return formatTime(statusData.value.data.connections[0]?.connected_at || '')
})

const lastMessageAt = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 'N/A'
  }
  return formatTime(statusData.value.data.connections[0]?.last_message_at || '')
})

const reconnectCount = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 0
  }
  return statusData.value.data.connections[0]?.reconnect_count || 0
})

const totalMessages = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return 0
  }
  return statusData.value.data.connections[0]?.messages_received || 0
})

const lastError = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return undefined
  }
  return statusData.value.data.connections[0]?.last_error
})

const isMessageStale = computed(() => {
  if (!statusData.value?.data.connections || statusData.value.data.connections.length === 0) {
    return false
  }
  const timestamp = statusData.value.data.connections[0]?.last_message_at
  return isStale(timestamp || '')
})

// Trading pairs subscribed on this connection
const tradingPairs = computed(() => {
  if (!statusData.value?.data.connections) return []
  return statusData.value.data.connections.map(conn => conn.instrument_id)
})

async function refreshStatus() {
  loading.value = true
  error.value = ''
  
  try {
    const data = await fetchWebSocketStatus()
    statusData.value = data
  } catch (err: any) {
    error.value = err.message || 'Failed to fetch WebSocket status'
    console.error('Error fetching WebSocket status:', err)
  } finally {
    loading.value = false
  }
}

function getStatusType(status: string): 'success' | 'warning' | 'danger' | 'info' {
  switch (status) {
    case 'connected':
      return 'success'
    case 'reconnecting':
      return 'warning'
    case 'disconnected':
      return 'danger'
    default:
      return 'info'
  }
}

function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleString()
  } catch {
    return timestamp
  }
}

function formatNumber(num: number): string {
  return num.toLocaleString()
}

function isStale(timestamp: string): boolean {
  try {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    return diff > 60000 // More than 1 minute
  } catch {
    return false
  }
}

function showConnectionDetails() {
  detailsDialogVisible.value = true
}

function startAutoRefresh() {
  if (!props.autoRefreshProp) return
  // Auto refresh every 5 seconds
  autoRefreshTimer = window.setInterval(() => {
    refreshStatus()
  }, 5000)
}

function stopAutoRefresh() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
}

// Watch for prop changes
watch(() => props.autoRefreshProp, (enabled) => {
  if (enabled) {
    refreshStatus()
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

onMounted(() => {
  refreshStatus()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.ws-monitor-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.error-message {
  margin-bottom: 16px;
}

.loading-container {
  padding: 20px 0;
}

.status-content {
  padding: 0;
}

.summary-row {
  margin-bottom: 20px;
}

.url-text {
  word-break: break-all;
  font-family: monospace;
}

.stale-message {
  color: #f56c6c;
  font-weight: 500;
}

.connection-details {
  padding: 10px 0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
}

.ws-info {
  margin-bottom: 20px;
}

:deep(.el-statistic__head) {
  font-size: 13px;
  color: #909399;
}

:deep(.el-statistic__number) {
  font-size: 24px;
  font-weight: 600;
}

:deep(.el-descriptions) {
  width: 100%;
}
</style>
