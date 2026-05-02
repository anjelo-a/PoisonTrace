import type {
  CandidateRecord,
  CounterpartyRecord,
  PagedResponse,
  TransactionRecord,
  WalletSyncRunSummary,
  IngestionRunSummary,
  OverviewMetrics,
} from "@poisontrace/contracts";

export type OverviewResponse = {
  metrics: OverviewMetrics & { passedTransactions: number; passRatePct: number };
  recentCandidates: CandidateRecord[];
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

export const apiClient = {
  getOverview: () => request<OverviewResponse>("/api/overview"),
  getCandidates: (page = 1, pageSize = 50) => request<PagedResponse<CandidateRecord>>(`/api/candidates?page=${page}&page_size=${pageSize}`),
  getTransactions: (page = 1, pageSize = 50) => request<PagedResponse<TransactionRecord>>(`/api/transactions?page=${page}&page_size=${pageSize}`),
  getRuns: (page = 1, pageSize = 50) => request<PagedResponse<IngestionRunSummary>>(`/api/runs?page=${page}&page_size=${pageSize}`),
  getWalletSync: (page = 1, pageSize = 50) => request<PagedResponse<WalletSyncRunSummary>>(`/api/wallet-sync?page=${page}&page_size=${pageSize}`),
  getCounterparties: (page = 1, pageSize = 50) => request<PagedResponse<CounterpartyRecord>>(`/api/counterparties?page=${page}&page_size=${pageSize}`),
  getSettings: () => request<SettingsResponse>("/api/settings"),
};
