import type {
  CandidateExplanation,
  CandidateListItem,
  CandidateRecord,
  CounterpartyListItem,
  PagedResponse,
  TransactionListItem,
  WalletInspectionSummary,
  WalletSyncRunListItem,
  IngestionRunListItem,
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

export type ExportGeneratedFile = {
  name: string;
  rowCount: number;
  sha256: string;
  downloadUrl: string;
};

export type ExportGenerateResponse = {
  runId: number;
  outDir: string;
  schemaVersion: string;
  generatedAt: string;
  files: ExportGeneratedFile[];
};

export type ExportListedFile = {
  name: string;
  sizeBytes: number;
  modifiedAt: string;
  downloadUrl: string;
};

export type ExportFilesResponse = {
  runId: number;
  files: ExportListedFile[];
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
  getCandidates: (page = 1, pageSize = 50) => request<PagedResponse<CandidateListItem>>(`/api/candidates?page=${page}&page_size=${pageSize}`),
  getTransactions: (page = 1, pageSize = 50) => request<PagedResponse<TransactionListItem>>(`/api/transactions?page=${page}&page_size=${pageSize}`),
  getRuns: (page = 1, pageSize = 50) => request<PagedResponse<IngestionRunListItem>>(`/api/runs?page=${page}&page_size=${pageSize}`),
  getWalletSync: (page = 1, pageSize = 50) => request<PagedResponse<WalletSyncRunListItem>>(`/api/wallet-sync?page=${page}&page_size=${pageSize}`),
  getCounterparties: (page = 1, pageSize = 50) => request<PagedResponse<CounterpartyListItem>>(`/api/counterparties?page=${page}&page_size=${pageSize}`),
  getCandidateExplanation: (walletSyncRunId: number, signature: string, transferIndex: number) =>
    request<CandidateExplanation>(`/api/candidates/${walletSyncRunId}/${encodeURIComponent(signature)}/${transferIndex}`),
  getCandidateReports: (runId: number, page = 1, pageSize = 50) =>
    request<PagedResponse<CandidateExplanation>>(`/api/reports/candidates?run_id=${runId}&page=${page}&page_size=${pageSize}`),
  getWalletReports: (runId: number, page = 1, pageSize = 50) =>
    request<PagedResponse<WalletInspectionSummary>>(`/api/reports/wallets?run_id=${runId}&page=${page}&page_size=${pageSize}`),
  generateExportDataset: (runId: number) =>
    request<ExportGenerateResponse>(`/api/exports/generate?run_id=${runId}`),
  getExportFiles: (runId: number) =>
    request<ExportFilesResponse>(`/api/exports/files?run_id=${runId}`),
  getSettings: () => request<SettingsResponse>("/api/settings"),
};
