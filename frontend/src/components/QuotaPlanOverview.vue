<script setup lang="ts">
import { nextTick, onBeforeUnmount, onBeforeUpdate, onMounted, onUpdated, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CodexPlanQuotaSummary, QuotaBucketSummary, QuotaValueByPlan } from '@/types'
import { formatDateTime } from '@/utils/format'
import { quotaAverageRemainingPercent, quotaCapacity, quotaMeterColor, quotaNormalizedFill } from '@/utils/quotas'

const props = defineProps<{
  plans: CodexPlanQuotaSummary[]
  quotaValueByPlan: QuotaValueByPlan
}>()

const { t } = useI18n()
const quotaValueElements = ref<HTMLElement[]>([])
let fitFrame: number | null = null
let valueResizeObserver: ResizeObserver | null = null

function bindQuotaValueElement(el: unknown) {
  if (el instanceof HTMLElement) {
    quotaValueElements.value.push(el)
  }
}

function cssPixelValue(element: HTMLElement, name: string, fallback: number) {
  const rawValue = window.getComputedStyle(element).getPropertyValue(name).trim()
  const parsed = Number.parseFloat(rawValue)
  return Number.isFinite(parsed) ? parsed : fallback
}

function fitQuotaValueElements() {
  for (const element of quotaValueElements.value) {
    const textElement = element.querySelector<HTMLElement>('.quota-bucket__value-text')
    if (!textElement) {
      continue
    }

    const availableWidth = Math.max(1, element.clientWidth)
    const maxFontSize = cssPixelValue(element, '--quota-value-max-size', 52)
    const minFontSize = cssPixelValue(element, '--quota-value-min-size', 18)
    element.style.setProperty('--quota-value-font-size', `${maxFontSize}px`)

    const textWidth = Math.max(1, textElement.scrollWidth)
    const fittedSize = Math.max(
      minFontSize,
      Math.min(maxFontSize, Math.floor((maxFontSize * (availableWidth / textWidth)) * 10) / 10),
    )
    element.style.setProperty('--quota-value-font-size', `${fittedSize}px`)
  }
}

function observeQuotaValueElements() {
  valueResizeObserver?.disconnect()
  valueResizeObserver = typeof ResizeObserver === 'undefined'
    ? null
    : new ResizeObserver(() => scheduleQuotaValueFit())
  if (!valueResizeObserver) {
    return
  }
  for (const element of quotaValueElements.value) {
    valueResizeObserver.observe(element)
  }
}

function scheduleQuotaValueFit() {
  if (fitFrame !== null) {
    window.cancelAnimationFrame(fitFrame)
  }
  fitFrame = window.requestAnimationFrame(() => {
    fitFrame = null
    observeQuotaValueElements()
    fitQuotaValueElements()
  })
}

function quotaUnitValue(planType: string, bucket: 'fiveHour' | 'weekly' | 'codeReviewWeekly') {
  if (bucket === 'codeReviewWeekly') {
    return 0
  }
  const planConfig = props.quotaValueByPlan[planType.toLowerCase()]
  return planConfig?.[bucket] ?? 0
}

function formatTotalRemainingValue(planType: string, bucket: 'fiveHour' | 'weekly' | 'codeReviewWeekly', value?: number | null) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return t('quotas.unavailable')
  }
  const unitValue = quotaUnitValue(planType, bucket)
  if (unitValue <= 0) {
    const roundedPercent = Math.abs(value - Math.round(value)) < 0.05 ? Math.round(value) : value.toFixed(1)
    return t('quotas.totalRemainingPercent', { value: roundedPercent })
  }
  const converted = (value / 100) * unitValue
  return t('quotas.totalRemainingValue', { value: formatMoneyValue(converted) })
}

function formatMoneyValue(value: number) {
  return value.toFixed(1)
}

function coverageLabel(successCount: number, failedCount: number) {
  return t('quotas.coverage', { success: successCount, total: successCount + failedCount })
}

function formatAverageRemaining(bucket: QuotaBucketSummary) {
  const average = quotaAverageRemainingPercent(bucket)
  if (typeof average !== 'number' || Number.isNaN(average)) {
    return t('quotas.unavailable')
  }
  const rounded = Math.abs(average - Math.round(average)) < 0.05 ? Math.round(average) : average.toFixed(1)
  return t('quotas.averageRemainingPercent', { value: rounded })
}

function formatCapacity(bucket: QuotaBucketSummary) {
  return t('quotas.capacityPercent', { value: quotaCapacity(bucket) })
}

function formatResetAt(value: string) {
  return value ? formatDateTime(value) : t('common.notAvailable')
}

onBeforeUpdate(() => {
  quotaValueElements.value = []
})

onUpdated(() => {
  scheduleQuotaValueFit()
})

onMounted(async () => {
  await nextTick()
  scheduleQuotaValueFit()
  window.addEventListener('resize', scheduleQuotaValueFit)
})

onBeforeUnmount(() => {
  if (fitFrame !== null) {
    window.cancelAnimationFrame(fitFrame)
    fitFrame = null
  }
  valueResizeObserver?.disconnect()
  valueResizeObserver = null
  window.removeEventListener('resize', scheduleQuotaValueFit)
})
</script>

<template>
  <section class="quota-grid">
    <article v-for="plan in plans" :key="plan.planType" class="panel quota-plan-card">
      <div class="panel-head panel-head--tight">
        <div>
          <p class="panel-kicker">{{ t('quotas.planLabel') }}</p>
          <h3>{{ plan.planType }}</h3>
        </div>
        <span class="quota-plan-card__count">
          {{ t('quotas.planAccounts', { count: plan.accountCount }) }}
        </span>
      </div>

      <div class="quota-plan-card__buckets quota-plan-card__buckets--visual">
        <section v-if="plan.fiveHour.supported" class="quota-bucket">
          <div class="quota-bucket__head">
            <strong>{{ t('quotas.buckets.fiveHour') }}</strong>
            <span>{{ coverageLabel(plan.fiveHour.successCount, plan.fiveHour.failedCount) }}</span>
          </div>
          <div :ref="bindQuotaValueElement" class="quota-bucket__value quota-bucket__value--hero">
            <span class="quota-bucket__value-text">{{ formatTotalRemainingValue(plan.planType, 'fiveHour', plan.fiveHour.totalRemainingPercent) }}</span>
          </div>
          <div class="quota-bucket__meter" aria-hidden="true">
            <span class="quota-bucket__meter-fill" :style="{ width: `${quotaNormalizedFill(plan.fiveHour)}%`, backgroundColor: quotaMeterColor(plan.fiveHour) }" />
          </div>
          <div class="quota-bucket__stats muted">
            <span>{{ formatAverageRemaining(plan.fiveHour) }}</span>
            <span>{{ formatCapacity(plan.fiveHour) }}</span>
          </div>
          <p class="muted quota-bucket__reset">
            {{ t('quotas.resetAt', { value: formatResetAt(plan.fiveHour.resetAt) }) }}
          </p>
        </section>

        <section class="quota-bucket">
          <div class="quota-bucket__head">
            <strong>{{ t('quotas.buckets.weekly') }}</strong>
            <span>{{ coverageLabel(plan.weekly.successCount, plan.weekly.failedCount) }}</span>
          </div>
          <div :ref="bindQuotaValueElement" class="quota-bucket__value quota-bucket__value--hero">
            <span class="quota-bucket__value-text">{{ formatTotalRemainingValue(plan.planType, 'weekly', plan.weekly.totalRemainingPercent) }}</span>
          </div>
          <div class="quota-bucket__meter" aria-hidden="true">
            <span class="quota-bucket__meter-fill" :style="{ width: `${quotaNormalizedFill(plan.weekly)}%`, backgroundColor: quotaMeterColor(plan.weekly) }" />
          </div>
          <div class="quota-bucket__stats muted">
            <span>{{ formatAverageRemaining(plan.weekly) }}</span>
            <span>{{ formatCapacity(plan.weekly) }}</span>
          </div>
          <p class="muted quota-bucket__reset">
            {{ t('quotas.resetAt', { value: formatResetAt(plan.weekly.resetAt) }) }}
          </p>
        </section>

        <section class="quota-bucket">
          <div class="quota-bucket__head">
            <strong>{{ t('quotas.buckets.codeReviewWeekly') }}</strong>
            <span>{{ coverageLabel(plan.codeReviewWeekly.successCount, plan.codeReviewWeekly.failedCount) }}</span>
          </div>
          <div :ref="bindQuotaValueElement" class="quota-bucket__value quota-bucket__value--hero">
            <span class="quota-bucket__value-text">{{ formatTotalRemainingValue(plan.planType, 'codeReviewWeekly', plan.codeReviewWeekly.totalRemainingPercent) }}</span>
          </div>
          <div class="quota-bucket__meter" aria-hidden="true">
            <span class="quota-bucket__meter-fill" :style="{ width: `${quotaNormalizedFill(plan.codeReviewWeekly)}%`, backgroundColor: quotaMeterColor(plan.codeReviewWeekly) }" />
          </div>
          <div class="quota-bucket__stats muted">
            <span>{{ formatAverageRemaining(plan.codeReviewWeekly) }}</span>
            <span>{{ formatCapacity(plan.codeReviewWeekly) }}</span>
          </div>
          <p class="muted quota-bucket__reset">
            {{ t('quotas.resetAt', { value: formatResetAt(plan.codeReviewWeekly.resetAt) }) }}
          </p>
        </section>
      </div>
    </article>
  </section>
</template>
