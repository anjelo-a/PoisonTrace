import { CheckCircle, XCircle } from "lucide-react";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../../lib/apiClient";
import { shortAddress, timeAgo } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Transactions() {
  const { page, pageSize, setPage, params, setParams } = useUrlPagination(1, 50);
  const relationFilter = params.get("relation") ?? "all";

  const { data, isLoading, error } = useQuery({
    queryKey: ["transactions", page, pageSize],
    queryFn: () => apiClient.getTransactions(page, pageSize),
    placeholderData: (prev) => prev,
  });

  const items = useMemo(() => {
    const rows = data?.items ?? [];
    if (relationFilter === "all") return rows;
    return rows.filter((row) => row.relationType === relationFilter);
  }, [data?.items, relationFilter]);

  const setRelation = (value: string) => {
    const next = new URLSearchParams(params);
    next.set("relation", value);
    next.set("page", "1");
    setParams(next, { replace: false });
  };

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border"><h1 className="text-2xl mb-2 tracking-tight">Transactions</h1><p className="text-muted-foreground text-sm">Normalization and eligibility outcomes</p></div>
      <div className="px-8 py-4 border-b border-border"><select value={relationFilter} onChange={(e) => setRelation(e.target.value)} className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground"><option value="all">All Relation Types</option><option value="receiver">Receiver</option><option value="sender">Sender</option></select></div>
      {isLoading ? <div className="px-8 py-4 text-sm text-muted-foreground">Loading transactions...</div> : null}
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">Failed to load transactions: {(error as Error).message}</div> : null}
      <div className="flex-1 overflow-auto">
        <table className="w-full"><thead className="bg-muted/30 border-b border-border sticky top-0"><tr><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Signature</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Time</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Norm</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Eligible</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Relation</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Dust</th><th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Amount</th></tr></thead><tbody className="divide-y divide-border">
          {items.map((tx) => <tr key={`${tx.signature}-${tx.transferIndex}-${tx.relationType}`} className="hover:bg-muted/30 transition-colors"><td className="px-8 py-5 font-mono text-sm">{shortAddress(tx.signature, 6, 6)}</td><td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(tx.blockTime)}</td><td className="px-8 py-5">{tx.normalizationStatus === "resolved" ? <div className="flex items-center gap-2"><CheckCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm">Resolved</span></div> : <div className="flex items-center gap-2"><XCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm text-destructive-foreground">{tx.normalizationStatus}</span></div>}</td><td className="px-8 py-5 text-sm">{tx.poisoningEligible ? "Yes" : "No"}</td><td className="px-8 py-5 text-sm text-muted-foreground">{tx.relationType}</td><td className="px-8 py-5 text-sm">{tx.dustStatus}</td><td className="px-8 py-5 font-mono text-sm text-muted-foreground">{tx.amountRaw}</td></tr>)}
        </tbody></table>
      </div>
      <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground"><div className="font-mono">page {page} • {data?.total ?? 0} total</div><div className="flex gap-3"><button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Previous</button><button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-4 py-2 border border-border text-xs disabled:opacity-50">Next</button></div></div>
    </div>
  );
}
