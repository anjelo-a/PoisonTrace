import type { RunStatus, WalletSyncStatus } from "@poisontrace/contracts";

type StatusTone = "default" | "secondary" | "outline" | "destructive";

const knownRunStatuses = new Set<RunStatus>([
  "running",
  "succeeded",
  "partially_succeeded",
  "failed",
  "timed_out",
  "cancelled",
]);

const knownWalletStatuses = new Set<WalletSyncStatus>([
  "queued",
  "running",
  "succeeded",
  "partial",
  "failed",
  "rate_limited",
  "timed_out",
  "skipped_invalid",
  "skipped_budget",
]);

export function parseRunStatus(status: string): RunStatus | "unknown" {
  return knownRunStatuses.has(status as RunStatus) ? (status as RunStatus) : "unknown";
}

export function parseWalletSyncStatus(status: string): WalletSyncStatus | "unknown" {
  return knownWalletStatuses.has(status as WalletSyncStatus) ? (status as WalletSyncStatus) : "unknown";
}

export function runStatusMeta(status: string): { label: string; tone: StatusTone } {
  const parsed = parseRunStatus(status);
  if (parsed === "unknown") {
    return { label: "Unknown", tone: "destructive" };
  }

  switch (parsed) {
    case "running":
      return { label: "Running", tone: "secondary" };
    case "succeeded":
      return { label: "Succeeded", tone: "default" };
    case "partially_succeeded":
      return { label: "Partially Succeeded", tone: "outline" };
    case "failed":
      return { label: "Failed", tone: "destructive" };
    case "timed_out":
      return { label: "Timed Out", tone: "destructive" };
    case "cancelled":
      return { label: "Cancelled", tone: "outline" };
  }
}

export function walletStatusMeta(status: string): { label: string; tone: StatusTone } {
  const parsed = parseWalletSyncStatus(status);
  if (parsed === "unknown") {
    return { label: "Unknown", tone: "destructive" };
  }

  switch (parsed) {
    case "queued":
      return { label: "Queued", tone: "secondary" };
    case "running":
      return { label: "Running", tone: "secondary" };
    case "succeeded":
      return { label: "Succeeded", tone: "default" };
    case "partial":
      return { label: "Partial", tone: "outline" };
    case "failed":
      return { label: "Failed", tone: "destructive" };
    case "rate_limited":
      return { label: "Rate Limited", tone: "outline" };
    case "timed_out":
      return { label: "Timed Out", tone: "destructive" };
    case "skipped_invalid":
      return { label: "Skipped Invalid", tone: "outline" };
    case "skipped_budget":
      return { label: "Skipped Budget", tone: "outline" };
  }
}

const nonCompletedRunStatuses = new Set<RunStatus>(["running"]);

export function isNonCompletedRunStatus(status: string): boolean {
  const parsed = parseRunStatus(status);
  return parsed !== "unknown" && nonCompletedRunStatuses.has(parsed);
}

const partialOrFailedRunStatuses = new Set<RunStatus>([
  "partially_succeeded",
  "failed",
  "timed_out",
  "cancelled",
]);

export function isPartialOrFailedRunStatus(status: string): boolean {
  const parsed = parseRunStatus(status);
  return parsed !== "unknown" && partialOrFailedRunStatuses.has(parsed);
}
