import { useMemo, useState } from "react";
import { X } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import type { CandidateListItem } from "@poisontrace/contracts";
import { apiClient } from "../../lib/apiClient";
import { formatDateTime, shortAddress, timeAgo } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Candidates() {
  const { page, pageSize, setPage, params, setParams } = useUrlPagination(1, 25);
  const [selected, setSelected] = useState<CandidateListItem | null>(null);

  const recencyFilter = params.get("recency") ?? "all";
  const sort = params.get("sort") ?? "latest";

  const { data, error, isLoading } = useQuery({
    queryKey: ["candidates", page, pageSize],
    queryFn: () => apiClient.getCandidates(page, pageSize),
    placeholderData: (prev) => prev,
  });

  const filtered = useMemo(() => {
    const items = [...(data?.items ?? [])];
    const byRecency = items.filter((item) => {
      if (recencyFilter === "lt1") return item.recencyDays < 1;
      if (recencyFilter === "lt7") return item.recencyDays < 7;
      return true;
    });
    byRecency.sort((a, b) => {
      if (sort === "oldest") return a.blockTime.localeCompare(b.blockTime);
      if (sort === "repeat") return b.repeatInjectionCount - a.repeatInjectionCount;
      return b.blockTime.localeCompare(a.blockTime);
    });
    return byRecency;
  }, [data?.items, recencyFilter, sort]);

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    next.set(key, value);
    next.set("page", "1");
    setParams(next, { replace: false });
  };

  return (
    <div className="h-full flex">
      <div className={`flex-1 flex flex-col ${selected ? "hidden md:flex" : ""}`}>
        <div className="p-8 border-b border-border">
          <h1 className="text-2xl mb-2 tracking-tight">Candidates</h1>
          <p className="text-muted-foreground text-sm">Emitted probable candidates requiring review</p>
        </div>

        <div className="px-8 py-4 border-b border-border flex gap-6">
          <select value={recencyFilter} onChange={(e) => setFilter("recency", e.target.value)} className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground">
            <option value="all">All Recency</option>
            <option value="lt1">&lt; 1 day</option>
            <option value="lt7">&lt; 7 days</option>
          </select>
          <select value={sort} onChange={(e) => setFilter("sort", e.target.value)} className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground">
            <option value="latest">Sort: Latest First</option>
            <option value="oldest">Sort: Oldest First</option>
            <option value="repeat">Sort: Repeat Count</option>
          </select>
        </div>

        {isLoading ? <div className="px-8 py-4 text-sm text-muted-foreground">Loading candidates...</div> : null}
        {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">Failed to load candidates: {(error as Error).message}</div> : null}

        <div className="flex-1 overflow-auto">
          <table className="w-full">
            <thead className="bg-muted/30 border-b border-border sticky top-0">
              <tr>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Signature</th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Block Time</th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Suspicious</th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Matched Legit</th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Repeat</th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Recency</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filtered.map((candidate) => (
                <tr key={`${candidate.signature}-${candidate.transferIndex}`} onClick={() => setSelected(candidate)} className={`cursor-pointer hover:bg-muted/30 transition-colors ${selected?.signature === candidate.signature ? "bg-muted/30" : ""}`}>
                  <td className="px-8 py-5"><div className="font-mono text-sm">{shortAddress(candidate.signature, 6, 6)}</div><div className="text-xs text-muted-foreground mt-1">{timeAgo(candidate.blockTime)}</div></td>
                  <td className="px-8 py-5 text-sm text-muted-foreground font-mono">{formatDateTime(candidate.blockTime)}</td>
                  <td className="px-8 py-5 font-mono text-sm">{shortAddress(candidate.suspiciousCounterparty, 6, 4)}</td>
                  <td className="px-8 py-5 font-mono text-sm text-destructive-foreground">{shortAddress(candidate.matchedLegitCounterparty, 6, 4)}</td>
                  <td className="px-8 py-5 text-sm font-mono">{candidate.repeatInjectionCount}</td>
                  <td className="px-8 py-5 text-sm text-muted-foreground">{candidate.recencyDays}d</td>
                </tr>
              ))}
              {filtered.length === 0 ? <tr><td className="px-8 py-5 text-sm text-muted-foreground" colSpan={6}>No candidates found for current filters.</td></tr> : null}
            </tbody>
          </table>
        </div>

        <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground">
          <div className="font-mono">Showing page {page} • {data?.total ?? 0} total</div>
          <div className="flex gap-3">
            <button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-4 py-2 border border-border hover:border-foreground disabled:opacity-50 text-xs">Previous</button>
            <button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-4 py-2 border border-border hover:border-foreground disabled:opacity-50 text-xs">Next</button>
          </div>
        </div>
      </div>

      {selected ? (
        <div className="w-full md:w-[560px] border-l border-border flex flex-col">
          <div className="p-8 border-b border-border flex items-start justify-between">
            <div><div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Candidate Detail</div><div className="font-mono text-sm break-all">{selected.signature}</div></div>
            <button onClick={() => setSelected(null)} className="ml-6 p-2 hover:bg-muted/30"><X className="w-4 h-4 text-muted-foreground" /></button>
          </div>
          <div className="flex-1 overflow-auto p-8 space-y-6 text-sm font-mono">
            <Row label="Block Time" value={formatDateTime(selected.blockTime)} />
            <Row label="Suspicious Counterparty" value={selected.suspiciousCounterparty} />
            <Row label="Matched Legit" value={selected.matchedLegitCounterparty} danger />
            <Row label="Repeat Injections" value={`${selected.repeatInjectionCount} events`} />
            <Row label="Recency" value={`${selected.recencyDays} days`} />
            <div className="text-xs text-muted-foreground">This view shows probable candidate evidence only. Unknown-gate blocked events are excluded from emitted candidates by contract.</div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function Row({ label, value, danger = false }: { label: string; value: string; danger?: boolean }) {
  return <div className="grid grid-cols-[180px_1fr]"><span className="text-muted-foreground">{label}:</span><span className={danger ? "text-destructive-foreground" : ""}>{value}</span></div>;
}
