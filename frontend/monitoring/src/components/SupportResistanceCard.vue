<template>
  <el-card shadow="hover" class="support-resistance-card">
    <template #header>
      <div class="card-header">
        <span class="card-title">
          <el-icon><Coordinate /></el-icon>
          Support & Resistance Levels
        </span>
        <el-tag v-if="data" size="small" type="info">
          {{ formatTime(data.analysis_time) }}
        </el-tag>
      </div>
    </template>

    <div v-if="loading" class="loading-container">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>Loading analysis data...</span>
    </div>

    <div v-else-if="error" class="error-container">
      <el-alert :title="error" type="error" :closable="false" />
    </div>

    <div v-else-if="!data" class="empty-container">
      <el-empty description="No data available" />
    </div>

    <div v-else class="levels-content">
      <div class="level-section resistance">
        <h4>Resistance Levels</h4>
        <div class="levels-grid">
          <div v-if="data.resistance_level_1" class="level-item">
            <span class="level-label">R1</span>
            <span class="level-value">{{ formatPrice(data.resistance_level_1) }}</span>
          </div>
          <div v-if="data.resistance_level_2" class="level-item">
            <span class="level-label">R2</span>
            <span class="level-value">{{ formatPrice(data.resistance_level_2) }}</span>
          </div>
          <div v-if="!data.resistance_level_1 && !data.resistance_level_2" class="no-data">
            No resistance levels detected
          </div>
        </div>
      </div>

      <el-divider />

      <div class="level-section support">
        <h4>Support Levels</h4>
        <div class="levels-grid">
          <div v-if="data.support_level_1" class="level-item">
            <span class="level-label">S1</span>
            <span class="level-value">{{ formatPrice(data.support_level_1) }}</span>
          </div>
          <div v-if="data.support_level_2" class="level-item">
            <span class="level-label">S2</span>
            <span class="level-value">{{ formatPrice(data.support_level_2) }}</span>
          </div>
          <div v-if="!data.support_level_1 && !data.support_level_2" class="no-data">
            No support levels detected
          </div>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { Coordinate, Loading } from '@element-plus/icons-vue'
import type { SupportResistanceData } from '../types/analysis'

defineProps<{
  data?: SupportResistanceData
  loading?: boolean
  error?: string
}>()

function formatPrice(price: string): string {
  const num = parseFloat(price)
  return isNaN(num) ? price : num.toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6
  })
}

function formatTime(time: string): string {
  try {
    const date = new Date(time)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return time
  }
}
</script>

<style scoped>
.support-resistance-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 16px;
}

.loading-container,
.error-container,
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  gap: 12px;
}

.levels-content {
  padding: 10px 0;
}

.level-section h4 {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #606266;
}

.level-section.resistance h4 {
  color: #f56c6c;
}

.level-section.support h4 {
  color: #67c23a;
}

.levels-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.level-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
  transition: all 0.3s;
}

.level-item:hover {
  background-color: #ecf5ff;
  transform: translateY(-2px);
}

.level-label {
  font-size: 12px;
  font-weight: 600;
  color: #909399;
  text-transform: uppercase;
}

.level-value {
  font-size: 20px;
  font-weight: 700;
  color: #303133;
}

.resistance .level-value {
  color: #f56c6c;
}

.support .level-value {
  color: #67c23a;
}

.no-data {
  color: #909399;
  font-size: 14px;
  font-style: italic;
}
</style>
