import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { computed } from 'vue'
import LogConsole from '@/components/LogConsole.vue'
import { shellModeKey } from '@/layout/shell'
import { createTestI18n } from '@/test/i18n'

describe('LogConsole', () => {
  it('renders incoming log rows', () => {
    const wrapper = mount(LogConsole, {
      global: {
        plugins: [createTestI18n()],
      },
      props: {
        entries: [
          {
            kind: 'scan',
            level: 'info',
            message: 'scan started',
            timestamp: '2026-03-07T12:00:00Z',
          },
        ],
      },
    })

    expect(wrapper.text()).toContain('scan started')
    expect(wrapper.text()).toContain('Scan')
  })

  it('switches to the compact layout when shell mode is injected', () => {
    const wrapper = mount(LogConsole, {
      global: {
        plugins: [createTestI18n()],
        provide: {
          [shellModeKey as symbol]: computed(() => 'compact'),
        },
      },
      props: {
        entries: [
          {
            kind: 'maintain',
            level: 'warning',
            message: 'maintenance paused',
            timestamp: '2026-03-07T13:00:00Z',
          },
        ],
      },
    })

    expect(wrapper.classes()).toContain('log-console--compact')
    expect(wrapper.text()).toContain('maintenance paused')
  })
})
