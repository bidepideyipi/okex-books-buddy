<template>
  <div class="layout-container">
    <el-container>
      <!-- Header -->
      <el-header class="app-header">
        <div class="header-content">
          <div class="logo">
            <el-icon :size="24"><TrendCharts /></el-icon>
            <span class="title">OKEx Order Book Monitor</span>
          </div>
          <div class="header-controls">
            <slot name="header-controls">
              <el-tag :type="isConnected ? 'success' : 'danger'" effect="dark">
                {{ isConnected ? 'Connected' : 'Disconnected' }}
              </el-tag>
            </slot>
          </div>
        </div>
      </el-header>

      <el-container class="body-container">
        <!-- Sidebar -->
        <el-aside width="250px" class="sidebar">
          <el-menu
            :default-active="activeMenu"
            class="sidebar-menu"
            @select="handleMenuSelect"
          >
            <el-menu-item index="ws-monitor">
              <el-icon><Connection /></el-icon>
              <span>WS Monitor</span>
            </el-menu-item>
            <el-menu-item index="dashboard">
              <el-icon><Odometer /></el-icon>
              <span>Dashboard</span>
            </el-menu-item>
            <el-menu-item index="support-resistance">
              <el-icon><Coordinate /></el-icon>
              <span>Support/Resistance</span>
            </el-menu-item>
            <el-menu-item index="large-orders">
              <el-icon><Histogram /></el-icon>
              <span>Large Orders</span>
            </el-menu-item>
          </el-menu>
        </el-aside>

        <!-- Main content -->
        <el-main class="main-content">
          <slot />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { TrendCharts, Odometer, Coordinate, Histogram, Connection } from '@element-plus/icons-vue'
import type { TradingPair } from '../types/analysis'

defineProps<{
  availablePairs: TradingPair[]
  isConnected?: boolean
}>()

const emit = defineEmits<{
  (e: 'pair-change', pair: string): void
  (e: 'menu-select', index: string): void
}>()

const selectedPair = ref('BTC-USDT')
const activeMenu = ref('ws-monitor')

function handlePairChange(pair: string) {
  emit('pair-change', pair)
}

function handleMenuSelect(index: string) {
  activeMenu.value = index
  emit('menu-select', index)
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background-color: #f5f7fa;
  position: fixed;
  top: 0;
  left: 0;
  margin: 0;
  padding: 0;
}

:deep(.el-container) {
  height: 100%;
  width: 100%;
}

:deep(.body-container) {
  height: calc(100vh - 60px);
  overflow: hidden;
}

.app-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 600;
}

.sidebar {
  background-color: white;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.05);
  height: 100%;
  overflow-y: auto;
}

.pair-selector {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
}

.pair-selector h3 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
}

.pair-select {
  width: 100%;
}

.pair-option {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pair-desc {
  font-size: 12px;
  color: #909399;
}

.sidebar-menu {
  border-right: none;
  padding-top: 20px;
}

.main-content {
  background-color: #f5f7fa;
  padding: 20px;
  height: 100%;
  overflow-y: auto;
  flex: 1;
}
</style>
