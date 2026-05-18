import { createI18n } from 'vue-i18n'
import { messages } from '@/i18n/messages'
import { detectPreferredLocale, localeEnglish, normalizeLocaleCode, type SupportedLocale } from '@/utils/locale'

export const i18n = createI18n({
  legacy: false,
  locale: detectPreferredLocale(),
  fallbackLocale: localeEnglish,
  messages,
})

export function setI18nLocale(locale: string): SupportedLocale {
  const next = normalizeLocaleCode(locale)
  i18n.global.locale.value = next
  return next
}
