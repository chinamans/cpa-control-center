import { describe, expect, it } from 'vitest'
import {
  compactHeightThreshold,
  compactWidthThreshold,
  resolveDashboardDrawerSize,
  resolveDonutLayoutMode,
  resolveShellMode,
  wideHeightThreshold,
  wideWidthThreshold,
} from '@/layout/shell'

describe('shell layout helpers', () => {
  it('resolves shell mode across wide, desktop, and compact thresholds', () => {
    expect(resolveShellMode(wideWidthThreshold, wideHeightThreshold)).toBe('wide')
    expect(resolveShellMode(compactWidthThreshold, compactHeightThreshold)).toBe('desktop')
    expect(resolveShellMode(compactWidthThreshold - 1, compactHeightThreshold)).toBe('compact')
    expect(resolveShellMode(compactWidthThreshold, compactHeightThreshold - 1)).toBe('compact')
  })

  it('returns drawer sizes that match each shell mode', () => {
    expect(resolveDashboardDrawerSize('wide')).toBe('min(1200px, calc(100vw - 48px))')
    expect(resolveDashboardDrawerSize('desktop')).toBe('min(1120px, calc(100vw - 32px))')
    expect(resolveDashboardDrawerSize('compact')).toBe('min(calc(100vw - 16px), 96vw)')
  })

  it('keeps the donut mode container-driven within the current shell mode', () => {
    expect(resolveDonutLayoutMode('compact', 560, 340)).toBe('compact')
    expect(resolveDonutLayoutMode('desktop', 500, 280)).toBe('desktop')
    expect(resolveDonutLayoutMode('wide', 500, 320)).toBe('wide')
    expect(resolveDonutLayoutMode('wide', 410, 320)).toBe('compact')
  })
})
