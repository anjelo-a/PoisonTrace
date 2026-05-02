export type RunStatus = "running" | "completed" | "failed" | "partial";

export type WalletSyncStatus = "running" | "completed" | "failed" | "partial";

export type NormalizationStatus =
  | "resolved"
  | "unresolved_owner"
  | "unsupported_asset";

export type RelationType = "sender" | "receiver";

export type UnknownGateReason =
  | "baseline_truncated"
  | "dust_threshold_missing"
  | "owner_unresolved"
  | "scan_window_incomplete"
  | "retry_exhausted"
  | "wallet_timeout"
  | "run_timeout"
  | "cancellation"
  | string;

export interface OverviewMetrics {
  candidatesEmitted: number;
  unknownGateBlocks: number;
  transactionsScanned: number;
  lastScanUpdateAt: string;
  scanWindowLabel: string;
}

export interface IngestionRunSummary {
  runId: number;
  startedAt: string;
  completedAt: string | null;
  status: RunStatus;
  walletsRequested: number;
  walletsCompleted: number;
  walletsPartial: number;
  walletsFailed: number;
  transactionsFetched: number;
  transactionsInserted: number;
  transactionsLinked: number;
  transactionsFailedToNormalize: number;
  poisoningCandidatesInserted: number;
  incompleteWindow: boolean;
  truncationReason: string | null;
}

export interface IngestionRunListItem {
  runId: number;
  status: RunStatus;
  startedAt: string;
  completedAt: string | null;
  walletsProcessed: number;
  walletsFailed: number;
  walletsSkipped: number;
  incompleteWindows: number;
  truncationWalletRate: string;
  unknownGatePresent: boolean;
}

export interface WalletSyncRunSummary {
  walletSyncRunId: number;
  runId: number;
  focalWallet: string;
  scanStartAt: string;
  scanEndAt: string;
  baselineStartAt: string;
  baselineEndAt: string;
  status: WalletSyncStatus;
  baselineComplete: boolean;
  incompleteWindow: boolean;
  unknownGateReason: UnknownGateReason | null;
  truncationReason: string | null;
  poisoningCandidatesInserted: number;
}

export interface WalletSyncRunListItem {
  walletSyncRunId: number;
  runId: number;
  focalWallet: string;
  status: WalletSyncStatus;
  baselineStartAt: string;
  baselineEndAt: string;
  scanStartAt: string;
  scanEndAt: string;
  baselineComplete: boolean;
  incompleteWindow: boolean;
  unknownGateReason: UnknownGateReason | null;
  transactionsFetched: number;
  truncationReason: string | null;
  updatedAt: string;
}

export interface CandidateRecord {
  walletSyncRunId: number;
  focalWallet: string;
  signature: string;
  transferIndex: number;
  blockTime: string;
  suspiciousCounterparty: string;
  matchedLegitCounterparty: string;
  repeatInjectionCount: number;
  lookalikePrefixMatch: number;
  lookalikeSuffixMatch: number;
  recencyDays: number;
  isDust: boolean;
  amountRaw: string;
  unknownGateReason: UnknownGateReason | null;
}

export interface CandidateListItem {
  walletSyncRunId: number;
  focalWallet: string;
  signature: string;
  transferIndex: number;
  blockTime: string;
  suspiciousCounterparty: string;
  matchedLegitCounterparty: string;
  repeatInjectionCount: number;
  recencyDays: number;
}

export interface TransactionRecord {
  signature: string;
  transferIndex: number;
  blockTime: string;
  assetType: "native_sol" | "spl_fungible" | "unsupported";
  normalizationStatus: NormalizationStatus;
  poisoningEligible: boolean;
  relationType: RelationType;
  sourceOwner: string | null;
  destinationOwner: string | null;
  fromTokenAccount: string | null;
  toTokenAccount: string | null;
  amountRaw: string;
  isDust: boolean | null;
  dustStatus?: string;
}

export interface TransactionListItem {
  focalWallet: string;
  signature: string;
  transferIndex: number;
  blockTime: string;
  normalizationStatus: NormalizationStatus;
  poisoningEligible: boolean;
  relationType: RelationType;
  dustStatus: string;
  amountRaw: string;
  assetType: "native_sol" | "spl_fungible" | "unsupported";
}

export interface CounterpartyRecord {
  focalWallet: string;
  counterpartyAddress: string;
  hasBaselineOutboundNonDust: boolean;
  baselineLastOutboundAt: string | null;
  firstSeenAt: string;
  lastSeenAt: string;
  inboundCount: number;
  outboundCount: number;
  isNewCounterparty: boolean | null;
  baselineComplete: boolean;
  candidateLinks?: number;
}

export interface CounterpartyListItem {
  focalWallet: string;
  counterpartyAddress: string;
  firstSeenAt: string;
  lastSeenAt: string;
  inboundCount: number;
  outboundCount: number;
  lastOutboundAt: string | null;
  candidateLinks: number;
}

export interface ExportManifestFile {
  name: string;
  rowCount: number;
  sha256: string;
}

export interface ExportManifest {
  schemaVersion: string;
  generatedAt: string;
  sourceFilters: {
    runId?: number;
    startedAtFrom?: string;
    startedAtTo?: string;
  };
  files: ExportManifestFile[];
}

export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
