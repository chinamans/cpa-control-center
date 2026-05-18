import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatusPill from '@/components/StatusPill.vue'
import { createTestI18n } from '@/test/i18n'

describe('StatusPill', () => {
  it('renders the state label', () => {
    const wrapper = mount(StatusPill, {
      global: {
        plugins: [createTestI18n()],
      },
      props: {
        state: 'Quota Limited',
      },
    })

    expect(wrapper.text()).toContain('Quota Limited')
  })
})
