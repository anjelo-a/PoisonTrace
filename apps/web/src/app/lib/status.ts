import type { RunStatus, WalletSyncStatus } from "@poisontrace/contracts";

type StatusTone = "default" | "secondary" | "outline" | "destructive";

export function runStatusMeta(status: RunStatus): { label: string; tone: StatusTone } {
  switch (status) {
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

export function walletStatusMeta(status: WalletSyncStatus): { label: string; tone: StatusTone } {
  switch (status) {
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

const partialOrFailedRunStatuses = new Set<RunStatus>([
  "partially_succeeded",
  "failed",
  "timed_out",
  "cancelled",
]);

export function isPartialOrFailedRunStatus(status: RunStatus): boolean {
  return partialOrFailedRunStatuses.has(status);
}

