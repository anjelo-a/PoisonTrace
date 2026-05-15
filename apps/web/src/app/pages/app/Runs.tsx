import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { IngestionRunListItem } from "@poisontrace/contracts";
import { Badge } from "../../components/ui/badge";
import { apiClient } from "../../lib/apiClient";
import { percentFromString, timeAgo } from "../../lib/format";
import { isPartialOrFailedRunStatus, runStatusMeta } from "../../lib/status";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Runs() {
  const { page, pageSize, setPage } = useUrlPagination(1, 25);
  const { data, isLoading, error } = useQuery({ queryKey: ["runs", page, pageSize], queryFn: () => apiClient.getRuns(page, pageSize), placeholderData: (prev) => prev });
  const stats = useMemo(() => {
    const rows = data?.items ?? [];
    return {
      total: rows.length,
      unknown: rows.filter((r) => Boolean(r.unknownGatePresent)).length,
      partialOrFailed: rows.filter((r) => isPartialOrFailedRunStatus(r.status)).length,
    };
  }, [data?.items]);

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border"><h1 className="text-2xl mb-2 tracking-tight">Detection Runs</h1><p className="text-muted-foreground text-sm">Bounded execution outcomes and unknown-gate context</p></div>
      <div className="px-8 py-6 border-b border-border grid grid-cols-3 gap-12"><Stat label="Loaded Runs" value={String(stats.total)} /><Stat label="Partial/Failed" value={String(stats.partialOrFailed)} /><Stat label="Unknown Gates" value={String(stats.unknown)} /></div>
      {isLoading ? <div className="px-8 py-4 text-sm text-muted-foreground">Loading runs...</div> : null}
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">Failed to load runs: {(error as Error).message}</div> : null}
      <div className="flex-1 overflow-auto"><table className="w-full"><thead className="bg-muted/30 border-b border-border sticky top-0"><tr><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Run</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Status</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Wallets P/F/S</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Incomplete</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Truncation</th></tr></thead><tbody className="divide-y divide-border">{(data?.items ?? []).map((run: IngestionRunListItem) => <tr key={run.runId} className="hover:bg-muted/30"><td className="px-8 py-5"><div className="font-mono text-sm">run-{run.runId}</div><div className="text-xs text-muted-foreground mt-1">{timeAgo(run.startedAt)}</div></td><td className="px-8 py-5 text-sm"><RunStatusBadge status={run.status} /></td><td className="px-8 py-5 text-sm font-mono">{run.walletsProcessed}/{run.walletsFailed}/{run.walletsSkipped}</td><td className="px-8 py-5 text-sm">{run.incompleteWindows}</td><td className="px-8 py-5 text-sm text-muted-foreground">{percentFromString(run.truncationWalletRate)}</td></tr>)}</tbody></table></div>
      <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground"><div className="font-mono">page {page} • {data?.total ?? 0} total</div><div className="flex gap-3"><button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Previous</button><button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Next</button></div></div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div><div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">{label}</div><div className="text-2xl font-mono tracking-tight">{value}</div></div>;
}

function RunStatusBadge({ status }: { status: IngestionRunListItem["status"] }) {
  const meta = runStatusMeta(status);
  return <Badge variant={meta.tone}>{meta.label}</Badge>;
}
