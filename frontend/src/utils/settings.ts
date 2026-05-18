import type { AppSettings, QuotaValueByPlan, ScheduleSettings } from '@/types'
import { detectPreferredLocale } from '@/utils/locale'

type Translate = (key: string, params?: Record<string, unknown>) => string

const fallbackTranslate: Translate = (key) => key
export const quotaValuePlanKeys = ['free', 'plus', 'pro', 'team', 'business', 'enterprise'] as const

export function createDefaultQuotaValueByPlan(): QuotaValueByPlan {
  return {
    free: { fiveHour: 0, weekly: 0 },
    plus: { fiveHour: 25, weekly: 100 },
    pro: { fiveHour: 0, weekly: 0 },
    team: { fiveHour: 0, weekly: 0 },
    business: { fiveHour: 0, weekly: 0 },
    enterprise: { fiveHour: 0, weekly: 0 },
  }
}

export function normalizeQuotaValueByPlan(input?: Partial<QuotaValueByPlan> | null): QuotaValueByPlan {
  const defaults = createDefaultQuotaValueByPlan()
  const normalized: QuotaValueByPlan = {}
  for (const plan of quotaValuePlanKeys) {
    normalized[plan] = {
      ...defaults[plan],
      ...(input?.[plan] ?? {}),
    }
  }
  for (const [plan, value] of Object.entries(input ?? {})) {
    if (!normalized[plan]) {
      normalized[plan] = {
        fiveHour: 0,
        weekly: 0,
      }
    }
    normalized[plan].fiveHour = normalizeQuotaValue(normalized[plan].fiveHour)
    normalized[plan].weekly = normalizeQuotaValue(normalized[plan].weekly)
  }
  return normalized
}

function normalizeQuotaValue(value: unknown): number {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? Math.max(0, numeric) : 0
}

export function createDefaultSettings(): AppSettings {
  return {
    baseUrl: '',
    managementToken: '',
    locale: detectPreferredLocale(),
    detailedLogs: false,
    targetType: 'codex',
    provider: '',
    scanStrategy: 'full',
    scanBatchSize: 1000,
    skipKnown401: true,
    probeWorkers: 40,
    actionWorkers: 20,
    quotaWorkers: 10,
    timeoutSeconds: 15,
    retries: 3,
    userAgent: 'codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal',
    quotaAction: 'disable',
    quotaCheckFree: false,
    quotaCheckPlus: true,
    quotaCheckPro: true,
    quotaCheckTeam: true,
    quotaCheckBusiness: true,
    quotaCheckEnterprise: true,
    quotaFreeMaxAccounts: 100,
    quotaValueByPlan: createDefaultQuotaValueByPlan(),
    quotaAutoRefreshEnabled: false,
    quotaAutoRefreshCron: '',
    delete401: true,
    autoReenable: true,
    exportDirectory: '',
    schedule: createDefaultScheduleSettings(),
  }
}

export function validateSettings(settings: AppSettings, t: Translate = fallbackTranslate): Record<string, string> {
  const errors: Record<string, string> = {}

  if (!settings.baseUrl.trim()) {
    errors.baseUrl = t('validation.baseUrlRequired')
  } else if (!/^https?:\/\//i.test(settings.baseUrl.trim())) {
    errors.baseUrl = t('validation.baseUrlProtocol')
  }

  if (!settings.managementToken.trim()) {
    errors.managementToken = t('validation.managementTokenRequired')
  }

  if (settings.probeWorkers < 1) {
    errors.probeWorkers = t('validation.probeWorkersMin')
  }
  if (settings.actionWorkers < 1) {
    errors.actionWorkers = t('validation.actionWorkersMin')
  }
  if (settings.quotaWorkers < 1) {
    errors.quotaWorkers = t('validation.quotaWorkersMin')
  }
  if (!['full', 'incremental'].includes(settings.scanStrategy)) {
    errors.scanStrategy = t('validation.scanStrategyInvalid')
  }
  if (settings.scanStrategy === 'incremental' && settings.scanBatchSize < 1) {
    errors.scanBatchSize = t('validation.scanBatchSizeMin')
  }
  if (settings.timeoutSeconds < 1) {
    errors.timeoutSeconds = t('validation.timeoutMin')
  }
  if (settings.retries < 0) {
    errors.retries = t('validation.retriesMin')
  }
  if (!['disable', 'delete'].includes(settings.quotaAction)) {
    errors.quotaAction = t('validation.quotaActionInvalid')
  }
  if (settings.quotaFreeMaxAccounts < -1) {
    errors.quotaFreeMaxAccounts = t('validation.quotaFreeMaxAccountsMin')
  }
  for (const [plan, value] of Object.entries(settings.quotaValueByPlan ?? {})) {
    if (!Number.isFinite(value.fiveHour) || !Number.isFinite(value.weekly) || value.fiveHour < 0 || value.weekly < 0) {
      errors[`quotaValueByPlan.${plan}`] = t('validation.quotaValueMin')
    }
  }
  if (settings.quotaAutoRefreshEnabled) {
    if (!settings.quotaAutoRefreshCron.trim()) {
      errors.quotaAutoRefreshCron = t('validation.quotaAutoRefreshCronRequired')
    } else if (!isValidCronExpression(settings.quotaAutoRefreshCron)) {
      errors.quotaAutoRefreshCron = t('validation.quotaAutoRefreshCronInvalid')
    }
  }
  if (settings.schedule.enabled) {
    if (!['scan', 'maintain'].includes(settings.schedule.mode)) {
      errors.scheduleMode = t('validation.scheduleModeInvalid')
    }
    if (!settings.schedule.cron.trim()) {
      errors.scheduleCron = t('validation.scheduleCronRequired')
    } else if (!isValidCronExpression(settings.schedule.cron)) {
      errors.scheduleCron = t('validation.scheduleCronInvalid')
    }
  }

  return errors
}

export function createDefaultScheduleSettings(): ScheduleSettings {
  return {
    enabled: false,
    mode: 'scan',
    cron: '',
  }
}

export function isValidCronExpression(value: string): boolean {
  const parts = value.trim().split(/\s+/)
  if (parts.length !== 5) {
    return false
  }

  const fieldSpecs: Array<{ min: number; max: number }> = [
    { min: 0, max: 59 },
    { min: 0, max: 23 },
    { min: 1, max: 31 },
    { min: 1, max: 12 },
    { min: 0, max: 7 },
  ]

  return parts.every((field, index) => isValidCronField(field, fieldSpecs[index].min, fieldSpecs[index].max))
}

export function cronMatchesDate(expression: string, date: Date): boolean {
  const parts = expression.trim().split(/\s+/)
  if (parts.length !== 5) {
    return false
  }

  const values = [
    date.getMinutes(),
    date.getHours(),
    date.getDate(),
    date.getMonth() + 1,
    date.getDay(),
  ]

  const fieldSpecs: Array<{ min: number; max: number }> = [
    { min: 0, max: 59 },
    { min: 0, max: 23 },
    { min: 1, max: 31 },
    { min: 1, max: 12 },
    { min: 0, max: 7 },
  ]

  return parts.every((field, index) => cronFieldMatches(field, values[index], fieldSpecs[index].min, fieldSpecs[index].max))
}

function isValidCronField(field: string, min: number, max: number): boolean {
  return field.split(',').every((segment) => isValidCronSegment(segment.trim(), min, max))
}

function cronFieldMatches(field: string, value: number, min: number, max: number): boolean {
  return field.split(',').some((segment) => cronSegmentMatches(segment.trim(), value, min, max))
}

function isValidCronSegment(segment: string, min: number, max: number): boolean {
  if (!segment) {
    return false
  }
  if (segment === '*') {
    return true
  }

  const [base, stepValue] = segment.split('/')
  if (segment.split('/').length > 2) {
    return false
  }
  if (stepValue !== undefined) {
    if (!/^\d+$/.test(stepValue)) {
      return false
    }
    const step = Number(stepValue)
    if (!Number.isInteger(step) || step <= 0) {
      return false
    }
  }

  if (base === '*') {
    return true
  }

  if (/^\d+$/.test(base)) {
    const value = Number(base)
    return value >= min && value <= max
  }

  const rangeMatch = base.match(/^(\d+)-(\d+)$/)
  if (!rangeMatch) {
    return false
  }

  const start = Number(rangeMatch[1])
  const end = Number(rangeMatch[2])
  return start >= min && end <= max && start <= end
}

function cronSegmentMatches(segment: string, value: number, min: number, max: number): boolean {
  if (!segment) {
    return false
  }
  if (segment === '*') {
    return true
  }

  const [base, stepValue] = segment.split('/')
  if (segment.split('/').length > 2) {
    return false
  }

  let baseMatches = false
  if (base === '*') {
    baseMatches = value >= min && value <= max
  } else if (/^\d+$/.test(base)) {
    baseMatches = Number(base) === value
  } else {
    const rangeMatch = base.match(/^(\d+)-(\d+)$/)
    if (!rangeMatch) {
      return false
    }
    const start = Number(rangeMatch[1])
    const end = Number(rangeMatch[2])
    if (start > end) {
      return false
    }
    baseMatches = value >= start && value <= end
  }

  if (!baseMatches) {
    return false
  }
  if (stepValue === undefined) {
    return true
  }
  if (!/^\d+$/.test(stepValue)) {
    return false
  }
  const step = Number(stepValue)
  if (!Number.isInteger(step) || step <= 0) {
    return false
  }
  if (base === '*') {
    return (value - min) % step === 0
  }
  if (/^\d+$/.test(base)) {
    return value === Number(base)
  }
  const rangeMatch = base.match(/^(\d+)-(\d+)$/)
  if (!rangeMatch) {
    return false
  }
  const start = Number(rangeMatch[1])
  return (value - start) % step === 0
}
