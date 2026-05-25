export type PagedResponse<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};

export type OverviewResponse = {
  metrics: {
    candidatesEmitted: number;
    unknownGateBlocks: number;
    transactionsScanned: number;
    passedTransactions: number;
    lastScanUpdateAt: string;
    scanWindowLabel: string;
    passRatePct: number;
  };
  recentCandidates: Array<{
    walletSyncRunId: number;
    focalWallet: string;
    signature: string;
    transferIndex: number;
    blockTime: string;
    suspiciousCounterparty: string;
    repeatInjectionCount: number;
    recencyDays: number;
  }>;
};

export type CandidateRow = {
  walletSyncRunId: number;
  focalWallet: string;
  signature: string;
  transferIndex: number;
  blockTime: string;
  suspiciousCounterparty: string;
  matchedLegitCounterparty: string;
  repeatInjectionCount: number;
  recencyDays: number;
  unknownGateReason: string;
};

export type TransactionRow = {
  focalWallet: string;
  signature: string;
  transferIndex: number;
  blockTime: string;
  normalizationStatus: string;
  poisoningEligible: boolean;
  relationType: string;
  dustStatus: string;
  amountRaw: string;
  assetType: string;
};

export type RunRow = {
  id: number;
  status: string;
  startedAt: string;
  completedAt: string;
  walletsProcessed: number;
  walletsFailed: number;
  walletsSkipped: number;
  incompleteWindows: number;
  truncationWalletRate: string;
  unknownGatePresent: boolean;
};

export type WalletSyncRow = {
  walletSyncRunId: number;
  ingestionRunId: number;
  focalWallet: string;
  status: string;
  baselineStartAt: string;
  baselineEndAt: string;
  scanStartAt: string;
  scanEndAt: string;
  baselineComplete: boolean;
  incompleteWindow: boolean;
  unknownGateReason: string;
  transactionsFetched: number;
  truncationReason: string;
  updatedAt: string;
};

export type CounterpartyRow = {
  focalWallet: string;
  counterpartyAddress: string;
  firstSeenAt: string;
  lastSeenAt: string;
  inboundCount: number;
  outboundCount: number;
  lastOutboundAt: string;
  candidateLinks: number;
};

export type ExportRow = {
  id: string;
  runId: number;
  timestamp: string;
  type: string;
  format: string;
  status: string;
};

export type SettingsResponse = {
  maxWalletsPerRun: number;
  maxTXPagesPerWallet: number;
  maxTXPerWallet: number;
  maxConcurrentWallets: number;
  walletSyncTimeoutSeconds: number;
  runTimeoutSeconds: number;
  maxHeliusRetries: number;
  heliusRequestDelayMS: number;
  baselineLookbackDays: number;
  scanWindowDays: number;
  lookalikeRecencyDays: number;
  lookalikePrefixMin: number;
  lookalikeSuffixMin: number;
  lookalikeSingleSideMin: number;
  minInjectionCount: number;
};

export type ManualRunStartRequest = {
  addresses: string;
  walletAddresses?: string[];
  scanStart?: string;
  scanEnd?: string;
  baselineLookbackDays?: number;
};

export type ManualRunStartResponse = {
  runId: number;
  walletCount: number;
  scanStart: string;
  scanEnd: string;
  baselineLookbackDays: number;
  status: "running";
};

const RAW_API_BASE = import.meta.env.VITE_API_BASE ?? import.meta.env.VITE_API_BASE_URL ?? "";
const API_BASE = RAW_API_BASE.replace(/\/+$/, "");
const DEFAULT_REQUEST_TIMEOUT_MS = 10_000;
const REQUEST_TIMEOUT_MS_ENV = Number(import.meta.env.VITE_API_TIMEOUT_MS);
const REQUEST_TIMEOUT_MS = Number.isFinite(REQUEST_TIMEOUT_MS_ENV) && REQUEST_TIMEOUT_MS_ENV > 0
  ? REQUEST_TIMEOUT_MS_ENV
  : DEFAULT_REQUEST_TIMEOUT_MS;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const timeoutId = globalThis.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const response = await fetch(`${API_BASE}${path}`, { ...init, signal: controller.signal });
    if (!response.ok) {
      let detail = "";
      try {
        const body = await response.json() as { error?: string };
        detail = body.error ? `: ${body.error}` : "";
      } catch {
        detail = "";
      }
      throw new Error(`request failed: ${response.status}${detail}`);
    }
    return response.json() as Promise<T>;
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new Error(`request timed out after ${REQUEST_TIMEOUT_MS}ms`);
    }
    throw error;
  } finally {
    globalThis.clearTimeout(timeoutId);
  }
}

export function getOverview() {
  return request<OverviewResponse>("/api/overview");
}

export function getCandidates(page = 1, pageSize = 50) {
  return request<PagedResponse<CandidateRow>>(`/api/candidates?page=${page}&page_size=${pageSize}`);
}

export function getTransactions(page = 1, pageSize = 50) {
  return request<PagedResponse<TransactionRow>>(`/api/transactions?page=${page}&page_size=${pageSize}`);
}

export function getRuns(page = 1, pageSize = 50) {
  return request<PagedResponse<RunRow>>(`/api/runs?page=${page}&page_size=${pageSize}`);
}

export function startManualRun(payload: ManualRunStartRequest) {
  return request<ManualRunStartResponse>("/api/runs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function getWalletSync(page = 1, pageSize = 50) {
  return request<PagedResponse<WalletSyncRow>>(`/api/wallet-sync?page=${page}&page_size=${pageSize}`);
}

export function getCounterparties(page = 1, pageSize = 50) {
  return request<PagedResponse<CounterpartyRow>>(`/api/counterparties?page=${page}&page_size=${pageSize}`);
}

export function getExports() {
  return request<{ items: ExportRow[] }>("/api/exports");
}

export function getSettings() {
  return request<SettingsResponse>("/api/settings");
}

export function updateSettings(payload: SettingsResponse) {
  return request<SettingsResponse>("/api/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
