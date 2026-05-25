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
  ManualRunStartRequest,
  ManualRunStartResponse,
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

export const apiClient = {
  getOverview: () => request<OverviewResponse>("/api/overview"),
  getCandidates: (page = 1, pageSize = 50) => request<PagedResponse<CandidateListItem>>(`/api/candidates?page=${page}&page_size=${pageSize}`),
  getTransactions: (page = 1, pageSize = 50) => request<PagedResponse<TransactionListItem>>(`/api/transactions?page=${page}&page_size=${pageSize}`),
  getRuns: (page = 1, pageSize = 50) => request<PagedResponse<IngestionRunListItem>>(`/api/runs?page=${page}&page_size=${pageSize}`),
  startManualRun: (payload: ManualRunStartRequest) =>
    request<ManualRunStartResponse>("/api/runs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),
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
  updateSettings: (payload: SettingsResponse) =>
    request<SettingsResponse>("/api/settings", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),
};
