export const localeEnglish = 'en-US' as const
export const localeChinese = 'zh-CN' as const

export type SupportedLocale = typeof localeEnglish | typeof localeChinese

export function normalizeLocaleCode(locale: string | null | undefined): SupportedLocale {
  const value = String(locale ?? '').trim().toLowerCase()
  if (value.startsWith('zh')) {
    return localeChinese
  }
  return localeEnglish
}

export function detectPreferredLocale(): SupportedLocale {
  if (typeof navigator !== 'undefined') {
    const preferred = navigator.language || navigator.languages?.[0] || ''
    return normalizeLocaleCode(preferred)
  }
  return localeEnglish
}
