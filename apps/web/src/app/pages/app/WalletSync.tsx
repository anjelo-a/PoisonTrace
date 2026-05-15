import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { WalletSyncRunListItem } from "@poisontrace/contracts";
import { Badge } from "../../components/ui/badge";
import { apiClient } from "../../lib/apiClient";
import { walletStatusMeta } from "../../lib/status";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function WalletSync() {
  const { page, pageSize, setPage } = useUrlPagination(1, 25);
  const settings = useQuery({ queryKey: ["settings"], queryFn: () => apiClient.getSettings() });
  const sync = useQuery({ queryKey: ["wallet-sync", page, pageSize], queryFn: () => apiClient.getWalletSync(page, pageSize), placeholderData: (prev) => prev });

  const incompleteMissingReason = useMemo(
    () => (sync.data?.items ?? []).filter((row) => row.incompleteWindow && !(row.unknownGateReason ?? "").trim()).length,
    [sync.data?.items],
  );

  return (
    <div className="p-8 max-w-6xl">
      <div className="mb-12"><h1 className="text-2xl mb-2 tracking-tight">Scan Configuration</h1><p className="text-muted-foreground text-sm">Read-only bounds plus wallet sync incomplete/unknown state evidence</p></div>

      <div className="border border-border mb-16 p-8 grid grid-cols-2 gap-6 text-sm font-mono">
        <KV k="Max Wallets per Run" v={String(settings.data?.maxWalletsPerRun ?? "-")} />
        <KV k="Max TX Pages per Wallet" v={String(settings.data?.maxTXPagesPerWallet ?? "-")} />
        <KV k="Max TX per Wallet" v={String(settings.data?.maxTXPerWallet ?? "-")} />
        <KV k="Max Concurrent Wallets" v={String(settings.data?.maxConcurrentWallets ?? "-")} />
      </div>

      {incompleteMissingReason > 0 ? <div className="mb-6 text-sm text-destructive-foreground">Data integrity warning: {incompleteMissingReason} incomplete wallet sync rows are missing unknown-gate reason.</div> : null}

      <div className="border border-border">
        <table className="w-full"><thead className="bg-muted/30 border-b border-border"><tr><th className="px-6 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Wallet</th><th className="px-6 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Status</th><th className="px-6 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Baseline Complete</th><th className="px-6 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Incomplete Window</th><th className="px-6 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Unknown Gate Reason</th></tr></thead><tbody className="divide-y divide-border">{(sync.data?.items ?? []).map((row) => <tr key={row.walletSyncRunId}><td className="px-6 py-4 font-mono text-sm">{row.focalWallet}</td><td className="px-6 py-4 text-sm"><WalletStatusBadge status={row.status} /></td><td className="px-6 py-4 text-sm">{String(row.baselineComplete)}</td><td className="px-6 py-4 text-sm">{String(row.incompleteWindow)}</td><td className="px-6 py-4 text-sm text-destructive-foreground">{row.unknownGateReason || "-"}</td></tr>)}</tbody></table>
      </div>

      <div className="mt-4 flex gap-3"><button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Previous</button><button onClick={() => setPage(page + 1)} disabled={Boolean(sync.data && page * pageSize >= sync.data.total)} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Next</button></div>
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return <div className="flex justify-between py-3 border-b border-border"><span className="text-muted-foreground">{k}:</span><span>{v}</span></div>;
}

function WalletStatusBadge({ status }: { status: WalletSyncRunListItem["status"] }) {
  const meta = walletStatusMeta(status);
  return <Badge variant={meta.tone}>{meta.label}</Badge>;
}
