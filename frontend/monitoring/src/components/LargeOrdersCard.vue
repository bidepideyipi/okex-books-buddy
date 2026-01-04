<template>
  <el-card shadow="hover" class="large-orders-card">
    <template #header>
      <div class="card-header">
        <span class="card-title">
          <el-icon><Histogram /></el-icon>
          Large Order Distribution
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

    <div v-else class="orders-content">
      <!-- Sentiment Badge -->
      <div class="sentiment-section">
        <h4>Market Sentiment</h4>
        <el-tag 
          :type="getSentimentType(data.large_order_trend)" 
          size="large"
          effect="dark"
          class="sentiment-tag"
        >
          {{ getSentimentLabel(data.large_order_trend) }}
        </el-tag>
        <div class="sentiment-details">
          <div class="sentiment-value">
            <span class="label">Sentiment:</span>
            <span class="value">{{ formatSentiment(data.sentiment) }}</span>
          </div>
          <div class="sentiment-indicator">
            <span class="label">Indicator:</span>
            <span class="value">{{ getSentimentInterpretation(data.sentiment) }}</span>
          </div>
        </div>
      </div>

      <el-divider />

      <!-- Order Distribution -->
      <div class="distribution-section">
        <div class="order-item buy">
          <div class="order-header">
            <el-icon :size="20"><TrendCharts /></el-icon>
            <span class="order-label">Large Buy Orders</span>
          </div>
          <div class="order-value">
            {{ formatNotional(data.large_buy_orders) }}
          </div>
          <div class="order-unit">USDT</div>
        </div>

        <div class="vs-divider">
          <el-icon :size="24"><Position /></el-icon>
        </div>

        <div class="order-item sell">
          <div class="order-header">
            <el-icon :size="20"><TrendCharts /></el-icon>
            <span class="order-label">Large Sell Orders</span>
          </div>
          <div class="order-value">
            {{ formatNotional(data.large_sell_orders) }}
          </div>
          <div class="order-unit">USDT</div>
        </div>
      </div>

      <!-- Balance Indicator -->
      <div class="balance-section">
        <div class="balance-bar">
          <div 
            class="balance-fill buy-fill" 
            :style="{ width: getBuyPercentage(data) + '%' }"
          />
          <div 
            class="balance-fill sell-fill" 
            :style="{ width: getSellPercentage(data) + '%' }"
          />
        </div>
        <div class="balance-labels">
          <span class="buy-label">{{ getBuyPercentage(data).toFixed(1) }}%</span>
          <span class="sell-label">{{ getSellPercentage(data).toFixed(1) }}%</span>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { Histogram, Loading, TrendCharts, Position } from '@element-plus/icons-vue'
import type { LargeOrderData } from '../types/analysis'

defineProps<{
  data?: LargeOrderData
  loading?: boolean
  error?: string
}>()

function formatNotional(value?: string): string {
  if (!value) return '0.00'
  const num = parseFloat(value)
  return isNaN(num) ? value : num.toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
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

function getSentimentType(trend?: string) {
  if (trend === 'bullish') return 'success'
  if (trend === 'bearish') return 'danger'
  return 'info'
}

function getSentimentLabel(trend?: string) {
  if (trend === 'bullish') return '🐂 Bullish'
  if (trend === 'bearish') return '🐻 Bearish'
  return '⚖️ Neutral'
}

function getBuyPercentage(data: LargeOrderData): number {
  const buy = parseFloat(data.large_buy_orders || '0')
  const sell = parseFloat(data.large_sell_orders || '0')
  const total = buy + sell
  return total > 0 ? (buy / total) * 100 : 50
}

function getSellPercentage(data: LargeOrderData): number {
  const buy = parseFloat(data.large_buy_orders || '0')
  const sell = parseFloat(data.large_sell_orders || '0')
  const total = buy + sell
  return total > 0 ? (sell / total) * 100 : 50
}

function formatSentiment(sentiment?: string): string {
  if (!sentiment) return '0.00'
  const num = parseFloat(sentiment)
  if (isNaN(num)) return sentiment
  return num.toFixed(3)
}

function getSentimentInterpretation(sentiment?: string): string {
  if (!sentiment) return 'Neutral'
  const num = parseFloat(sentiment)
  if (isNaN(num)) return 'Unknown'
  
  if (num > 0.3) return 'Bullish (Strong buying pressure)'
  if (num < -0.3) return 'Bearish (Strong selling pressure)'
  return 'Neutral (Balanced market)'
}
</script>

<style scoped>
.large-orders-card {
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

.orders-content {
  padding: 10px 0;
}

.sentiment-section {
  text-align: center;
  margin-bottom: 20px;
}

.sentiment-section h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #606266;
}

.sentiment-tag {
  font-size: 18px;
  padding: 12px 24px;
  margin-bottom: 12px;
}

.sentiment-details {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.sentiment-value,
.sentiment-indicator {
  display: flex;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
}

.sentiment-value .label,
.sentiment-indicator .label {
  font-weight: 600;
  color: #909399;
}

.sentiment-value .value,
.sentiment-indicator .value {
  font-weight: 500;
  color: #303133;
}

.distribution-section {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 20px;
  align-items: center;
  margin: 20px 0;
}

.order-item {
  padding: 20px;
  border-radius: 8px;
  transition: all 0.3s;
}

.order-item.buy {
  background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
  border: 2px solid #67c23a;
}

.order-item.sell {
  background: linear-gradient(135deg, #ffebee 0%, #ffcdd2 100%);
  border: 2px solid #f56c6c;
}

.order-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
}

.order-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.order-label {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
}

.order-value {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 4px;
}

.order-item.buy .order-value {
  color: #67c23a;
}

.order-item.sell .order-value {
  color: #f56c6c;
}

.order-unit {
  font-size: 12px;
  color: #909399;
}

.vs-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
}

.balance-section {
  margin-top: 24px;
}

.balance-bar {
  display: flex;
  height: 20px;
  border-radius: 10px;
  overflow: hidden;
  background-color: #f0f0f0;
}

.balance-fill {
  transition: width 0.5s ease;
}

.buy-fill {
  background: linear-gradient(90deg, #67c23a 0%, #85ce61 100%);
}

.sell-fill {
  background: linear-gradient(90deg, #f56c6c 0%, #f78989 100%);
}

.balance-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
}

.buy-label {
  color: #67c23a;
}

.sell-label {
  color: #f56c6c;
}
</style>
