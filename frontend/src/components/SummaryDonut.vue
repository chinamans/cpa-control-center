<script setup lang="ts">
import { PieChart, type PieSeriesOption } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  type TitleComponentOption,
  type TooltipComponentOption,
} from 'echarts/components'
import { getInstanceByDom, init, use, type ComposeOption, type ECharts } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { computed, inject, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DashboardSummary } from '@/types'
import { emitDebug, emitDebugError } from '@/utils/debug'
import { resolveDonutLayoutMode, shellModeKey, type ShellMode } from '@/layout/shell'

use([PieChart, TitleComponent, TooltipComponent, CanvasRenderer])

type DonutChartOption = ComposeOption<
  PieSeriesOption |
  TitleComponentOption |
  TooltipComponentOption
>

type DonutItem = {
  key: string
  value: number
  name: string
  color: string
}

const { t } = useI18n()

const props = defineProps<{
  summary: DashboardSummary
}>()

const shell = ref<HTMLDivElement | null>(null)
const chartRoot = ref<HTMLDivElement | null>(null)
const shellWidth = ref(0)
const shellHeight = ref(0)
const injectedShellMode = inject(shellModeKey, null)
let chart: ECharts | null = null
let resizeObserver: ResizeObserver | null = null
let renderFrame = 0

const items = computed<DonutItem[]>(() => [
  { key: 'pending', value: props.summary.pendingCount, name: t('states.pending'), color: '#7c9eb2' },
  { key: 'normal', value: props.summary.normalCount, name: t('states.normal'), color: '#2f7d61' },
  { key: 'invalid_401', value: props.summary.invalid401Count, name: t('states.invalid_401'), color: '#c2410c' },
  { key: 'quota_limited', value: props.summary.quotaLimitedCount, name: t('states.quota_limited'), color: '#d97706' },
  { key: 'recovered', value: props.summary.recoveredCount, name: t('states.recovered'), color: '#3b82f6' },
  { key: 'error', value: props.summary.errorCount, name: t('states.error'), color: '#6b7280' },
].filter((item) => item.value > 0))

const displayItems = computed<DonutItem[]>(() => {
  if (items.value.length > 0) {
    return items.value
  }
  return [
    {
      key: 'not_available',
      value: 0,
      name: t('common.notAvailable'),
      color: '#d6d0c0',
    },
  ]
})

const shellMode = computed<ShellMode>(() => injectedShellMode?.value ?? 'desktop')
const layoutMode = computed<ShellMode>(() => resolveDonutLayoutMode(shellMode.value, shellWidth.value, shellHeight.value))

const shellStyle = computed(() => ({
  '--donut-gap': layoutMode.value === 'compact' ? '0.62rem' : layoutMode.value === 'wide' ? '0.92rem' : '0.78rem',
}))

const chartPixelSize = computed(() => {
  const width = shellWidth.value
  const height = shellHeight.value
  const baseSize = layoutMode.value === 'wide' ? 230 : layoutMode.value === 'desktop' ? 204 : 172
  const horizontalBudget = Math.max(148, width - (layoutMode.value === 'compact' ? 28 : 48))
  const verticalReserve = layoutMode.value === 'compact' ? 118 : 146
  const verticalBudget = Math.max(148, height - verticalReserve)

  return `${Math.max(148, Math.min(baseSize, horizontalBudget, verticalBudget))}px`
})

const chartStyle = computed(() => ({
  '--donut-chart-size': chartPixelSize.value,
}))

const legendColumns = computed(() => {
  const total = Math.max(displayItems.value.length, 1)
  if (layoutMode.value === 'compact' || shellWidth.value < 380) {
    return 1
  }
  return Math.min(total, 2)
})

const legendStyle = computed(() => ({
  gridTemplateColumns: `repeat(${legendColumns.value}, minmax(0, 1fr))`,
  '--legend-gap': layoutMode.value === 'compact' ? '0.56rem' : layoutMode.value === 'wide' ? '0.78rem' : '0.66rem',
  '--legend-item-padding': layoutMode.value === 'compact' ? '0.56rem 0.68rem' : layoutMode.value === 'wide' ? '0.72rem 0.88rem' : '0.64rem 0.78rem',
  '--legend-name-size': layoutMode.value === 'compact' ? '0.8rem' : layoutMode.value === 'wide' ? '0.9rem' : '0.85rem',
  '--legend-value-size': layoutMode.value === 'compact' ? '1rem' : layoutMode.value === 'wide' ? '1.2rem' : '1.08rem',
  '--legend-swatch-size': layoutMode.value === 'compact' ? '10px' : '12px',
  '--legend-max-width': layoutMode.value === 'wide' ? '420px' : layoutMode.value === 'compact' ? '100%' : '388px',
}))

function chartMetrics(width: number, height: number) {
  const basis = Math.min(width, height)
  const compact = layoutMode.value === 'compact' || basis < 168
  const dense = basis < 144
  const wide = layoutMode.value === 'wide' && !compact

  return {
    radius: dense ? ['62%', '82%'] : compact ? ['64%', '84%'] : wide ? ['69%', '88%'] : ['66%', '86%'],
    center: ['50%', '47%'],
    titleTop: dense ? '38%' : '39%',
    subTitleTop: dense ? '55%' : '54%',
    titleSize: dense ? 16 : compact ? 19 : wide ? 28 : 23,
    subTitleSize: dense ? 8 : wide ? 11 : 10,
    borderWidth: dense ? 3 : 4,
    emphasisScale: dense ? 4 : wide ? 7 : 5,
  }
}

function updateBounds() {
  if (!shell.value) {
    return
  }
  shellWidth.value = shell.value.clientWidth
  shellHeight.value = shell.value.clientHeight
}

function renderChart() {
  if (!chartRoot.value) {
    emitDebug('summary-donut', 'render skipped: no chart root')
    return
  }
  if (chartRoot.value.clientWidth <= 0 || chartRoot.value.clientHeight <= 0) {
    emitDebug('summary-donut', 'render skipped: zero-size container', {
      width: chartRoot.value.clientWidth,
      height: chartRoot.value.clientHeight,
    })
    return
  }

  try {
    if (!chart) {
      chart = getInstanceByDom(chartRoot.value) ?? init(chartRoot.value)
    } else {
      chart.resize()
    }

    const metrics = chartMetrics(chartRoot.value.clientWidth, chartRoot.value.clientHeight)
    const seriesData = items.value.length > 0
      ? items.value.map(({ color, name, value }) => ({
          value,
          name,
          itemStyle: { color },
        }))
      : [{ value: 1, name: t('common.notAvailable'), itemStyle: { color: '#d6d0c0' } }]

    chart.setOption<DonutChartOption>({
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
      },
      title: [
        {
          text: `${props.summary.filteredAccounts || 0}`,
          left: 'center',
          top: metrics.titleTop,
          textStyle: {
            color: '#201b14',
            fontSize: metrics.titleSize,
            fontWeight: 800,
            lineHeight: metrics.titleSize + 4,
          },
        },
        {
          text: t('dashboard.trackedAccounts'),
          left: 'center',
          top: metrics.subTitleTop,
          textStyle: {
            color: '#7a705f',
            fontSize: metrics.subTitleSize,
            fontWeight: 700,
          },
        },
      ],
      series: [
        {
          type: 'pie',
          radius: metrics.radius,
          center: metrics.center,
          silent: false,
          label: {
            show: false,
          },
          labelLine: {
            show: false,
          },
          itemStyle: {
            borderColor: '#f5efe2',
            borderWidth: metrics.borderWidth,
          },
          emphasis: {
            scale: true,
            scaleSize: metrics.emphasisScale,
          },
          data: seriesData,
        },
      ],
    })
    emitDebug('summary-donut', 'rendered', {
      width: chartRoot.value.clientWidth,
      height: chartRoot.value.clientHeight,
      items: items.value.map((item) => ({ key: item.key, value: item.value })),
      filtered: props.summary.filteredAccounts,
    })
  } catch (error) {
    emitDebugError('summary-donut', 'render failed', error)
    console.error('SummaryDonut render failed', error)
  }
}

function queueRender() {
  if (renderFrame) {
    window.cancelAnimationFrame(renderFrame)
  }
  renderFrame = window.requestAnimationFrame(() => {
    renderFrame = 0
    renderChart()
  })
}

watch([items, () => props.summary.filteredAccounts], async () => {
  await nextTick()
  queueRender()
}, { deep: true })

onMounted(() => {
  emitDebug('summary-donut', 'mounted')
  updateBounds()
  queueRender()
  if (shell.value) {
    resizeObserver = new ResizeObserver(() => {
      updateBounds()
      queueRender()
    })
    resizeObserver.observe(shell.value)
  }
})

onActivated(() => {
  emitDebug('summary-donut', 'activated')
  updateBounds()
  queueRender()
})

onDeactivated(() => {
  emitDebug('summary-donut', 'deactivated')
  if (renderFrame) {
    window.cancelAnimationFrame(renderFrame)
    renderFrame = 0
  }
})

onBeforeUnmount(() => {
  emitDebug('summary-donut', 'before unmount')
  resizeObserver?.disconnect()
  resizeObserver = null
  if (renderFrame) {
    window.cancelAnimationFrame(renderFrame)
    renderFrame = 0
  }
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div
    ref="shell"
    class="summary-donut"
    :class="{
      'summary-donut--wide': layoutMode === 'wide',
      'summary-donut--desktop': layoutMode === 'desktop',
      'summary-donut--compact': layoutMode === 'compact',
    }"
    :style="shellStyle"
  >
    <div ref="chartRoot" class="summary-donut__chart" :style="chartStyle" />
    <div class="summary-donut__legend" :style="legendStyle">
      <div v-for="item in displayItems" :key="item.key" class="summary-donut__legend-item">
        <span class="summary-donut__legend-swatch" :style="{ backgroundColor: item.color }" />
        <span class="summary-donut__legend-name">{{ item.name }}</span>
        <strong class="summary-donut__legend-value">{{ item.value }}</strong>
      </div>
    </div>
  </div>
</template>

<style scoped>
.summary-donut {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--donut-gap, 0.8rem);
  width: 100%;
  height: 100%;
  min-height: 0;
  margin: 0 auto;
  padding: 0.1rem 0 0.35rem;
}

.summary-donut__chart {
  width: var(--donut-chart-size, 204px);
  height: var(--donut-chart-size, 204px);
  flex: none;
  margin-inline: auto;
}

.summary-donut__legend {
  display: grid;
  gap: var(--legend-gap, 0.75rem);
  width: min(100%, var(--legend-max-width, 388px));
  align-content: start;
}

.summary-donut__legend-item {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
  padding: var(--legend-item-padding, 0.7rem 0.8rem);
  border-radius: 16px;
  background: rgba(245, 239, 226, 0.88);
  border: 1px solid rgba(38, 34, 27, 0.08);
}

.summary-donut__legend-swatch {
  flex: none;
  width: var(--legend-swatch-size, 12px);
  height: var(--legend-swatch-size, 12px);
  border-radius: 999px;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.72);
}

.summary-donut__legend-name {
  flex: 1 1 auto;
  min-width: 0;
  color: #4e4639;
  font-size: var(--legend-name-size, 0.92rem);
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: normal;
  line-height: 1.2;
}

.summary-donut__legend-value {
  color: #1f1a14;
  font-size: var(--legend-value-size, 1.25rem);
  font-weight: 800;
  line-height: 1;
}

.summary-donut--compact .summary-donut__legend {
  align-self: stretch;
  width: 100%;
  grid-template-columns: 1fr !important;
}

.summary-donut--wide .summary-donut__legend-item {
  border-radius: 18px;
}
</style>
