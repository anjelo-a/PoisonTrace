import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../../lib/apiClient";
import { formatDateTime, shortAddress } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Counterparties() {
  const { page, pageSize, setPage, params, setParams } = useUrlPagination(1, 25);
  const runId = Number(params.get("run_id") ?? "0");

  const { data, isLoading, error } = useQuery({
    queryKey: ["wallet-reports", runId, page, pageSize],
    queryFn: () => apiClient.getWalletReports(runId, page, pageSize),
    enabled: runId > 0,
    placeholderData: (prev) => prev,
  });

  return (
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-2xl mb-2 tracking-tight">Wallet Inspection Summary</h1>
        <p className="text-muted-foreground text-sm">Run-scoped inspection status, unknown-gate context, and candidate totals per wallet</p>
      </div>

      <div className="mb-6">
        <label className="text-xs text-muted-foreground uppercase tracking-widest font-mono">Run ID</label>
        <div className="mt-2">
          <input
            value={runId > 0 ? String(runId) : ""}
            onChange={(e) => {
              const next = new URLSearchParams(params);
              next.set("run_id", e.target.value);
              next.set("page", "1");
              setParams(next, { replace: false });
            }}
            className="px-4 py-2 bg-transparent border border-border text-sm font-mono"
            placeholder="e.g. 42"
          />
        </div>
      </div>

      {runId <= 0 ? <div className="text-sm text-muted-foreground">Enter a run ID to view wallet-level inspection summary.</div> : null}
      {isLoading ? <div className="text-sm text-muted-foreground">Loading wallet summaries...</div> : null}
      {error ? <div className="text-sm text-destructive-foreground">Failed to load wallet summaries: {(error as Error).message}</div> : null}

      {runId > 0 ? (
        <div className="border border-border overflow-auto">
          <table className="w-full">
            <thead className="bg-muted/30 border-b border-border">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Wallet</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Candidates</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Unknown Blocks</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Incomplete</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Unknown Reason</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Scan End</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {(data?.items ?? []).map((item) => (
                <tr key={`${item.walletSyncRunId}`}>
                  <td className="px-4 py-3 text-sm font-mono">{shortAddress(item.focalWallet, 8, 6)}</td>
                  <td className="px-4 py-3 text-sm font-mono">{item.candidateCount}</td>
                  <td className="px-4 py-3 text-sm font-mono">{item.unknownGateBlockCount}</td>
                  <td className="px-4 py-3 text-sm">{item.incompleteWindow ? "yes" : "no"}</td>
                  <td className="px-4 py-3 text-sm font-mono text-muted-foreground">{item.unknownGateReason || ""}</td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">{formatDateTime(item.scanEndAt)}</td>
                </tr>
              ))}
              {(data?.items?.length ?? 0) === 0 ? <tr><td className="px-4 py-4 text-sm text-muted-foreground" colSpan={6}>No rows for selected run.</td></tr> : null}
            </tbody>
          </table>
        </div>
      ) : null}

      {runId > 0 ? (
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <div className="font-mono">Showing page {page} • {data?.total ?? 0} total</div>
          <div className="flex gap-2">
            <button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-3 py-2 border border-border text-xs disabled:opacity-50">Previous</button>
            <button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-3 py-2 border border-border text-xs disabled:opacity-50">Next</button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
