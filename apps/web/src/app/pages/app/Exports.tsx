import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { apiClient } from "../../lib/apiClient";
import { formatDateTime, shortAddress } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Exports() {
  const { page, pageSize, setPage, params, setParams } = useUrlPagination(1, 25);
  const runId = Number(params.get("run_id") ?? "0");

  const { data, isLoading, error } = useQuery({
    queryKey: ["candidate-reports", runId, page, pageSize],
    queryFn: () => apiClient.getCandidateReports(runId, page, pageSize),
    enabled: runId > 0,
    placeholderData: (prev) => prev,
  });

  return (
    <div className="p-8 max-w-7xl">
      <div className="mb-8">
        <h1 className="text-2xl mb-2 tracking-tight">Candidate Report Exports</h1>
        <p className="text-muted-foreground text-sm">Deterministic candidate evidence view for run-scoped JSONL/CSV exports</p>
      </div>

      <div className="mb-6">
        <label className="text-xs text-muted-foreground uppercase tracking-widest font-mono">Run ID</label>
        <div className="mt-2 flex gap-3">
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

      {runId <= 0 ? <div className="text-sm text-muted-foreground">Enter a run ID to inspect candidate evidence rows.</div> : null}
      {isLoading ? <div className="text-sm text-muted-foreground">Loading report rows...</div> : null}
      {error ? <div className="text-sm text-destructive-foreground">Failed to load report rows: {(error as Error).message}</div> : null}

      {runId > 0 ? (
        <div className="border border-border">
          <table className="w-full">
            <thead className="bg-muted/30 border-b border-border">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Wallet</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Signature</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Block Time</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Suspicious</th>
                <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Legit</th>
                <th className="px-4 py-3 text-right text-xs font-mono uppercase tracking-widest text-muted-foreground">Detail</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {(data?.items ?? []).map((item) => (
                <tr key={`${item.walletSyncRunId}-${item.signature}-${item.transferIndex}`} className="hover:bg-muted/30">
                  <td className="px-4 py-3 text-sm font-mono">{shortAddress(item.focalWallet, 6, 4)}</td>
                  <td className="px-4 py-3 text-sm font-mono">
                    <Link
                      className="hover:underline"
                      to={`/app/candidates?wallet_sync_run_id=${item.walletSyncRunId}&signature=${encodeURIComponent(item.signature)}&transfer_index=${item.transferIndex}`}
                    >
                      {shortAddress(item.signature, 8, 8)}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">{formatDateTime(item.blockTime)}</td>
                  <td className="px-4 py-3 text-sm font-mono">{shortAddress(item.suspiciousCounterparty, 6, 4)}</td>
                  <td className="px-4 py-3 text-sm font-mono text-destructive-foreground">{shortAddress(item.matchedLegitCounterparty, 6, 4)}</td>
                  <td className="px-4 py-3 text-right text-xs">
                    <Link
                      className="text-muted-foreground hover:text-foreground hover:underline"
                      to={`/app/candidates?wallet_sync_run_id=${item.walletSyncRunId}&signature=${encodeURIComponent(item.signature)}&transfer_index=${item.transferIndex}`}
                    >
                      Open Detail
                    </Link>
                  </td>
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
