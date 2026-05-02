import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../../lib/apiClient";
import { shortAddress, timeAgo } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Counterparties() {
  const { page, pageSize, setPage } = useUrlPagination(1, 50);
  const { data, isLoading, error } = useQuery({ queryKey: ["counterparties", page, pageSize], queryFn: () => apiClient.getCounterparties(page, pageSize), placeholderData: (prev) => prev });

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border"><h1 className="text-2xl mb-2 tracking-tight">Counterparties</h1><p className="text-muted-foreground text-sm">Wallet interaction history</p></div>
      {isLoading ? <div className="px-8 py-4 text-sm text-muted-foreground">Loading counterparties...</div> : null}
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">Failed to load counterparties: {(error as Error).message}</div> : null}
      <div className="flex-1 overflow-auto"><table className="w-full"><thead className="bg-muted/30 border-b border-border sticky top-0"><tr><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Address</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">First Seen</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Last Seen</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Inbound</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Outbound</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Candidate Links</th></tr></thead><tbody className="divide-y divide-border">{(data?.items ?? []).map((cp) => <tr key={`${cp.focalWallet}-${cp.counterpartyAddress}`}><td className="px-8 py-5 font-mono text-sm">{shortAddress(cp.counterpartyAddress, 6, 4)}</td><td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(cp.firstSeenAt)}</td><td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(cp.lastSeenAt)}</td><td className="px-8 py-5 text-sm font-mono">{cp.inboundCount}</td><td className="px-8 py-5 text-sm font-mono">{cp.outboundCount}</td><td className="px-8 py-5 text-sm">{cp.candidateLinks}</td></tr>)}</tbody></table></div>
      <div className="px-8 py-4 border-t border-border flex gap-3"><button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Previous</button><button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Next</button></div>
    </div>
  );
}
