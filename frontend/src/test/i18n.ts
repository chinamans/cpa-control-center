import { createI18n } from 'vue-i18n'
import { messages } from '@/i18n/messages'

export function createTestI18n(locale: 'en-US' | 'zh-CN' = 'en-US') {
  return createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'en-US',
    messages,
  })
}
