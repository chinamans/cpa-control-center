import type {
  AccountFilter,
  AccountPage,
  AccountRecord,
  ActionResult,
  AppSettings,
  BulkAccountActionResult,
  CodexQuotaSnapshot,
  ConnectionResult,
  DashboardSnapshot,
  DashboardSummary,
  ExportDownload,
  ExportResult,
  InventorySyncResult,
  MaintainOptions,
  MaintainResult,
  ScanDetail,
  ScanDetailPage,
  ScanSummary,
  SchedulerStatus,
} from '@/types'

interface WailsResponse<T> {
  result?: T
  error?: string
}

async function invoke<T>(method: string, args: unknown[] = []): Promise<T> {
  const response = await fetch(`/api/wails/${method}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ args }),
  })

  let payload: WailsResponse<T> | null = null
  try {
    payload = await response.json() as WailsResponse<T>
  } catch {
    // Leave payload empty and use the HTTP status text below.
  }

  if (!response.ok || payload?.error) {
    throw new Error(payload?.error || response.statusText || `Request failed: ${method}`)
  }

  return payload?.result as T
}

export function GetSettings(): Promise<AppSettings> {
  return invoke<AppSettings>('GetSettings')
}

export function SaveSettings(input: unknown): Promise<AppSettings> {
  return invoke<AppSettings>('SaveSettings', [input])
}

export function TestConnection(input: unknown): Promise<ConnectionResult> {
  return invoke<ConnectionResult>('TestConnection', [input])
}

export function TestAndSaveSettings(input: unknown): Promise<ConnectionResult> {
  return invoke<ConnectionResult>('TestAndSaveSettings', [input])
}

export function SyncInventory(): Promise<InventorySyncResult> {
  return invoke<InventorySyncResult>('SyncInventory')
}

export function GetSchedulerStatus(): Promise<SchedulerStatus> {
  return invoke<SchedulerStatus>('GetSchedulerStatus')
}

export function GetDashboardSummary(): Promise<DashboardSummary> {
  return invoke<DashboardSummary>('GetDashboardSummary')
}

export function GetDashboardSnapshot(): Promise<DashboardSnapshot> {
  return invoke<DashboardSnapshot>('GetDashboardSnapshot')
}

export function GetCodexQuotaSnapshot(): Promise<CodexQuotaSnapshot> {
  return invoke<CodexQuotaSnapshot>('GetCodexQuotaSnapshot')
}

export function GetCachedCodexQuotaSnapshot(): Promise<CodexQuotaSnapshot> {
  return invoke<CodexQuotaSnapshot>('GetCachedCodexQuotaSnapshot')
}

export function ListAccounts(filter: AccountFilter): Promise<AccountRecord[]> {
  return invoke<AccountRecord[]>('ListAccounts', [filter])
}

export function ListAccountsPage(filter: AccountFilter, page: number, pageSize: number): Promise<AccountPage> {
  return invoke<AccountPage>('ListAccountsPage', [filter, page, pageSize])
}

export function RunScan(): Promise<ScanSummary> {
  return invoke<ScanSummary>('RunScan')
}

export function CancelScan(): Promise<boolean> {
  return invoke<boolean>('CancelScan')
}

export function RunMaintain(options: MaintainOptions): Promise<MaintainResult> {
  return invoke<MaintainResult>('RunMaintain', [options])
}

export function ProbeAccount(name: string): Promise<AccountRecord> {
  return invoke<AccountRecord>('ProbeAccount', [name])
}

export function ProbeAccounts(names: string[]): Promise<BulkAccountActionResult> {
  return invoke<BulkAccountActionResult>('ProbeAccounts', [names])
}

export function SetAccountDisabled(name: string, disabled: boolean): Promise<ActionResult> {
  return invoke<ActionResult>('SetAccountDisabled', [name, disabled])
}

export function SetAccountsDisabled(names: string[], disabled: boolean): Promise<BulkAccountActionResult> {
  return invoke<BulkAccountActionResult>('SetAccountsDisabled', [names, disabled])
}

export function DeleteAccount(name: string): Promise<ActionResult> {
  return invoke<ActionResult>('DeleteAccount', [name])
}

export function DeleteAccounts(names: string[]): Promise<BulkAccountActionResult> {
  return invoke<BulkAccountActionResult>('DeleteAccounts', [names])
}

export function ExportAccounts(kind: string, format: string, path: string): Promise<ExportResult> {
  return invoke<ExportResult>('ExportAccounts', [kind, format, path])
}

export function ExportAccountsDownload(kind: string, format: string): Promise<ExportDownload> {
  return invoke<ExportDownload>('ExportAccountsDownload', [kind, format])
}

export function ListScanHistory(limit: number): Promise<ScanSummary[]> {
  return invoke<ScanSummary[]>('ListScanHistory', [limit])
}

export function GetScanDetails(runID: number): Promise<ScanDetail> {
  return invoke<ScanDetail>('GetScanDetails', [runID])
}

export function GetScanDetailsPage(runID: number, page: number, pageSize: number): Promise<ScanDetailPage> {
  return invoke<ScanDetailPage>('GetScanDetailsPage', [runID, page, pageSize])
}
