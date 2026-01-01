<template>
  <div class="connection-status">
    <el-dropdown trigger="hover">
      <el-tag 
        :type="statusType" 
        effect="dark"
        class="status-tag"
      >
        <el-icon class="status-icon" :class="{ 'is-loading': isConnecting }">
          <component :is="statusIcon" />
        </el-icon>
        <span>{{ statusText }}</span>
      </el-tag>
      
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item disabled>
            <div class="status-details">
              <div class="detail-row">
                <span class="label">Status:</span>
                <span class="value">{{ statusText }}</span>
              </div>
              <div class="detail-row">
                <span class="label">Mode:</span>
                <span class="value" :class="connectionModeClass">{{ connectionMode }}</span>
              </div>
              <div v-if="lastUpdate" class="detail-row">
                <span class="label">Last Update:</span>
                <span class="value">{{ lastUpdateTime }}</span>
              </div>
              <div v-if="reconnectAttempt > 0" class="detail-row">
                <span class="label">Reconnect:</span>
                <span class="value reconnect-value">Attempt {{ reconnectAttempt }}</span>
              </div>
            </div>
          </el-dropdown-item>
          <el-dropdown-item divided>
            <div class="info-text">
              <el-icon><InfoFilled /></el-icon>
              <span v-if="connectionMode === 'websocket'">
                Real-time updates via WebSocket
              </span>
              <span v-else>
                HTTP polling every 2 seconds
              </span>
            </div>
          </el-dropdown-item>
          <el-dropdown-item divided v-if="!isConnected" @click="handleReconnect">
            <el-icon><Refresh /></el-icon>
            Reconnect Now
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Connection, CircleClose, Loading, Warning, Refresh, InfoFilled } from '@element-plus/icons-vue'

const props = defineProps<{
  isConnected: boolean
  isConnecting?: boolean
  connectionMode: 'websocket' | 'polling'
  lastUpdate?: Date
  reconnectAttempt?: number
}>()

const emit = defineEmits<{
  (e: 'reconnect'): void
}>()

const statusType = computed(() => {
  if (props.isConnecting) return 'warning'
  if (props.isConnected) {
    return props.connectionMode === 'websocket' ? 'success' : 'info'
  }
  return 'danger'
})

const statusText = computed(() => {
  if (props.isConnecting) return 'Connecting...'
  if (props.isConnected) {
    return props.connectionMode === 'websocket' ? 'Live' : 'Polling'
  }
  return 'Disconnected'
})

const statusIcon = computed(() => {
  if (props.isConnecting) return Loading
  if (props.isConnected) return Connection
  if (props.reconnectAttempt && props.reconnectAttempt > 0) return Warning
  return CircleClose
})

const lastUpdateTime = computed(() => {
  if (!props.lastUpdate) return 'Never'
  const now = new Date()
  const diff = now.getTime() - props.lastUpdate.getTime()
  
  if (diff < 1000) return 'Just now'
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  return props.lastUpdate.toLocaleTimeString()
})

const connectionModeClass = computed(() => {
  return props.connectionMode === 'websocket' ? 'mode-websocket' : 'mode-polling'
})

function handleReconnect() {
  emit('reconnect')
}
</script>

<style scoped>
.connection-status {
  display: flex;
  align-items: center;
}

.status-tag {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 13px;
  user-select: none;
  transition: all 0.3s;
}

.status-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.status-icon {
  display: flex;
  align-items: center;
}

.status-icon.is-loading {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.status-details {
  padding: 8px 4px;
  min-width: 200px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 13px;
}

.detail-row .label {
  color: #909399;
  font-weight: 500;
  margin-right: 12px;
}

.detail-row .value {
  color: #303133;
  font-weight: 600;
}

.detail-row .value.mode-websocket {
  color: #67c23a;
}

.detail-row .value.mode-polling {
  color: #409eff;
}

.detail-row .value.reconnect-value {
  color: #e6a23c;
}

.info-text {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #606266;
  padding: 4px 0;
}

.info-text .el-icon {
  color: #409eff;
}
</style>
