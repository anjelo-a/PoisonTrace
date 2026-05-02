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

const API_BASE = import.meta.env.VITE_API_BASE ?? "";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`);
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  return response.json() as Promise<T>;
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
