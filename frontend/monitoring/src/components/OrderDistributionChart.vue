<template>
  <div class="chart-card">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><DataAnalysis /></el-icon>
            Order Distribution Analysis
          </span>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>Loading chart data...</span>
      </div>

      <div v-else-if="!hasData" class="empty-container">
        <el-empty description="No data available for visualization" />
      </div>

      <div v-else ref="chartRef" class="chart-container" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { DataAnalysis, Loading } from '@element-plus/icons-vue'
import type { LargeOrderData, SupportResistanceData } from '../types/analysis'

const props = defineProps<{
  largeOrderData?: LargeOrderData
  supportResistanceData?: SupportResistanceData
  loading?: boolean
}>()

const chartRef = ref<HTMLElement>()
let chartInstance: echarts.ECharts | null = null

const hasData = ref(false)

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)
  updateChart()
}

function updateChart() {
  if (!chartInstance || !props.largeOrderData) return

  const buyOrders = parseFloat(props.largeOrderData.large_buy_orders || '0')
  const sellOrders = parseFloat(props.largeOrderData.large_sell_orders || '0')

  hasData.value = buyOrders > 0 || sellOrders > 0

  if (!hasData.value) return

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} USDT ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      textStyle: {
        fontSize: 14
      }
    },
    series: [
      {
        name: 'Order Distribution',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: true,
          fontSize: 14,
          fontWeight: 'bold',
          formatter: '{b}\n{d}%'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: true
        },
        data: [
          {
            value: buyOrders,
            name: 'Buy Orders',
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: '#85ce61' },
                { offset: 1, color: '#67c23a' }
              ])
            }
          },
          {
            value: sellOrders,
            name: 'Sell Orders',
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: '#f78989' },
                { offset: 1, color: '#f56c6c' }
              ])
            }
          }
        ]
      }
    ]
  }

  chartInstance.setOption(option)
}

function resizeChart() {
  chartInstance?.resize()
}

onMounted(() => {
  initChart()
  window.addEventListener('resize', resizeChart)
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeChart)
  chartInstance?.dispose()
})

watch(
  () => [props.largeOrderData, props.loading],
  () => {
    if (!props.loading && chartInstance) {
      updateChart()
    }
  },
  { deep: true }
)
</script>

<style scoped>
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
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  gap: 12px;
  min-height: 400px;
}

.chart-container {
  width: 100%;
  height: 400px;
}
</style>
