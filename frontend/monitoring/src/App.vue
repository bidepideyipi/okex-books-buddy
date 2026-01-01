<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElNotification } from 'element-plus'
import Layout from './components/Layout.vue'
import ConnectionStatus from './components/ConnectionStatus.vue'
import SupportResistanceCard from './components/SupportResistanceCard.vue'
import LargeOrdersCard from './components/LargeOrdersCard.vue'
import OrderDistributionChart from './components/OrderDistributionChart.vue'
import WebSocketMonitor from './components/WebSocketMonitor.vue'
import { fetchAnalysisData, getAvailablePairs } from './services/api'
import { createAnalysisWebSocket, type WSMessage } from './services/websocket'
import type { AnalysisResponse } from './types/analysis'

const availablePairs = getAvailablePairs()
const selectedPair = ref('BTC-USDT')
const analysisData = ref<AnalysisResponse | null>(null)
const isLoading = ref(false)
const error = ref('')
const isConnected = ref(false)
const connectionMode = ref<'websocket' | 'polling'>('polling')
const lastUpdate = ref<Date>()
const reconnectAttempt = ref(0)
const currentView = ref<'dashboard' | 'support-resistance' | 'large-orders' | 'ws-monitor'>('ws-monitor')
const autoRefreshEnabled = ref(true)

let pollingInterval: number | null = null
const ws = createAnalysisWebSocket()

// Try WebSocket connection first, fallback to polling if it fails
async function initializeConnection() {
  try {
    // Setup WebSocket handlers
    ws.onConnect(() => {
      console.log('WebSocket connected, switching to real-time mode')
      connectionMode.value = 'websocket'
      isConnected.value = true
      reconnectAttempt.value = 0
      
      // Stop polling when WebSocket is connected
      stopPolling()
      
      // Subscribe to current pair
      ws.subscribe(selectedPair.value)
      
      ElNotification({
        title: 'Connected',
        message: 'Real-time updates enabled',
        type: 'success',
        duration: 2000
      })
    })

    ws.onDisconnect(() => {
      console.log('WebSocket disconnected, falling back to polling')
      isConnected.value = false
      
      // Fallback to polling when WebSocket disconnects
      startPolling()
    })

    ws.onError((err) => {
      console.error('WebSocket error:', err)
      reconnectAttempt.value++
    })

    ws.onMessage((message: WSMessage) => {
      if (message.type === 'analysis_update' && message.instrument_id === selectedPair.value) {
        // Update analysis data from WebSocket message
        if (message.data) {
          analysisData.value = {
            instrument_id: message.instrument_id,
            support_resistance: message.data.support_resistance,
            large_orders: message.data.large_orders
          }
          lastUpdate.value = new Date()
        }
      }
    })

    // Attempt WebSocket connection
    ws.connect()
    
    // Start with polling until WebSocket connects
    await loadAnalysisData()
    startPolling()
  } catch (err) {
    console.error('Failed to initialize connection:', err)
    // Fallback to polling only
    startPolling()
  }
}

async function loadAnalysisData() {
  if (!selectedPair.value) return
  
  isLoading.value = true
  error.value = ''
  
  try {
    const data = await fetchAnalysisData(selectedPair.value)
    analysisData.value = data
    lastUpdate.value = new Date()
    if (connectionMode.value === 'polling') {
      isConnected.value = true
    }
  } catch (err: any) {
    error.value = err.message || 'Failed to load analysis data'
    if (connectionMode.value === 'polling') {
      isConnected.value = false
    }
    console.error('Error fetching analysis data:', err)
  } finally {
    isLoading.value = false
  }
}

function handlePairChange(pair: string) {
  // Unsubscribe from old pair
  if (ws.isConnected()) {
    ws.unsubscribe(selectedPair.value)
  }
  
  selectedPair.value = pair
  
  // Subscribe to new pair
  if (ws.isConnected()) {
    ws.subscribe(pair)
  } else {
    // Load data immediately if using polling
    loadAnalysisData()
  }
}

function startPolling() {
  if (pollingInterval) return
  
  connectionMode.value = 'polling'
  pollingInterval = window.setInterval(() => {
    loadAnalysisData()
  }, 2000)
}

function stopPolling() {
  if (pollingInterval) {
    clearInterval(pollingInterval)
    pollingInterval = null
  }
}

function handleReconnect() {
  if (!ws.isConnected()) {
    ws.connect()
  }
}

function switchView(view: 'dashboard' | 'support-resistance' | 'large-orders' | 'ws-monitor') {
  currentView.value = view
}

function toggleAutoRefresh(enabled: boolean) {
  // This will be passed to WebSocketMonitor component
  autoRefreshEnabled.value = enabled
}

onMounted(() => {
  initializeConnection()
})

onUnmounted(() => {
  stopPolling()
  ws.disconnect()
})
</script>

<template>
  <Layout 
    :available-pairs="availablePairs" 
    :is-connected="isConnected"
    @pair-change="handlePairChange"
    @menu-select="switchView"
  >
    <template #header-controls>
      <div class="header-controls">
        <el-switch
          v-model="autoRefreshEnabled"
          active-text="Auto Refresh"
          inactive-text="Paused"
          @change="toggleAutoRefresh"
        />
      </div>
    </template>

    <!-- Dashboard View -->
    <div v-if="currentView === 'dashboard'" class="page-content">
      <h2 class="page-title">
        {{ selectedPair }} Analysis Dashboard
      </h2>

      <el-row :gutter="20" class="content-row">
        <el-col :xs="24" :sm="24" :md="12" :lg="12">
          <SupportResistanceCard
            :data="analysisData?.support_resistance"
            :loading="isLoading"
            :error="error"
          />
        </el-col>
        <el-col :xs="24" :sm="24" :md="12" :lg="12">
          <LargeOrdersCard
            :data="analysisData?.large_orders"
            :loading="isLoading"
            :error="error"
          />
        </el-col>
      </el-row>

      <el-row :gutter="20" class="content-row">
        <el-col :span="24">
          <OrderDistributionChart
            :large-order-data="analysisData?.large_orders"
            :support-resistance-data="analysisData?.support_resistance"
            :loading="isLoading"
          />
        </el-col>
      </el-row>
    </div>

    <!-- Support/Resistance View -->
    <div v-else-if="currentView === 'support-resistance'" class="page-content">
      <h2 class="page-title">
        {{ selectedPair }} Support/Resistance
      </h2>
      <el-row :gutter="20" class="content-row">
        <el-col :span="24">
          <SupportResistanceCard
            :data="analysisData?.support_resistance"
            :loading="isLoading"
            :error="error"
          />
        </el-col>
      </el-row>
    </div>

    <!-- Large Orders View -->
    <div v-else-if="currentView === 'large-orders'" class="page-content">
      <h2 class="page-title">
        {{ selectedPair }} Large Orders
      </h2>
      <el-row :gutter="20" class="content-row">
        <el-col :span="24">
          <LargeOrdersCard
            :data="analysisData?.large_orders"
            :loading="isLoading"
            :error="error"
          />
        </el-col>
      </el-row>
    </div>

    <!-- WebSocket Monitor View -->
    <div v-else-if="currentView === 'ws-monitor'" class="page-content">
      <h2 class="page-title">
        WebSocket Connection Monitor
      </h2>

      <el-row :gutter="20" class="content-row">
        <el-col :span="24">
          <WebSocketMonitor :auto-refresh-prop="autoRefreshEnabled" />
        </el-col>
      </el-row>
    </div>
  </Layout>
</template>

<style scoped>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-content {
  padding: 0;
  width: 100%;
}

.page-title {
  margin: 0 0 20px 0;
  color: #303133;
  font-size: 24px;
  font-weight: 600;
}

.content-row {
  margin-bottom: 20px;
}
</style>
