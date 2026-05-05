import {
  isNonCompletedRunStatus,
  isPartialOrFailedRunStatus,
  parseRunStatus,
  parseWalletSyncStatus,
  runStatusMeta,
  walletStatusMeta,
} from "../app/lib/status";

describe("status mapping", () => {
  it("parses known and unknown run statuses", () => {
    expect(parseRunStatus("running")).toBe("running");
    expect(parseRunStatus("cancelled")).toBe("cancelled");
    expect(parseRunStatus("future_status")).toBe("unknown");
  });

  it("parses known and unknown wallet sync statuses", () => {
    expect(parseWalletSyncStatus("queued")).toBe("queued");
    expect(parseWalletSyncStatus("partial")).toBe("partial");
    expect(parseWalletSyncStatus("mystery")).toBe("unknown");
  });

  it("maps known and unknown statuses to UI badges", () => {
    expect(runStatusMeta("succeeded")).toEqual({ label: "Succeeded", tone: "default" });
    expect(runStatusMeta("weird")).toEqual({ label: "Unknown", tone: "destructive" });
    expect(walletStatusMeta("timed_out")).toEqual({ label: "Timed Out", tone: "destructive" });
    expect(walletStatusMeta("weird")).toEqual({ label: "Unknown", tone: "destructive" });
  });

  it("handles non-completed and partial/failed semantics", () => {
    expect(isNonCompletedRunStatus("running")).toBe(true);
    expect(isNonCompletedRunStatus("succeeded")).toBe(false);
    expect(isNonCompletedRunStatus("unknown_value")).toBe(false);
    expect(isPartialOrFailedRunStatus("partially_succeeded")).toBe(true);
    expect(isPartialOrFailedRunStatus("running")).toBe(false);
    expect(isPartialOrFailedRunStatus("unknown_value")).toBe(false);
  });
});
