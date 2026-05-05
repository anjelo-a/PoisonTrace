import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../../lib/apiClient";
import { formatDateTime, shortAddress } from "../../lib/format";
import { useUrlPagination } from "../../lib/useUrlPagination";

export default function Counterparties() {
  const { page, pageSize, setPage } = useUrlPagination(1, 25);

  const { data, isLoading, error } = useQuery({
    queryKey: ["counterparties", page, pageSize],
    queryFn: () => apiClient.getCounterparties(page, pageSize),
    placeholderData: (prev) => prev,
  });

  return (
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-2xl mb-2 tracking-tight">Counterparties</h1>
        <p className="text-muted-foreground text-sm">Wallet-level counterparty relationships sourced from normalized transfer history</p>
      </div>

      {isLoading ? <div className="text-sm text-muted-foreground">Loading counterparties...</div> : null}
      {error ? <div className="text-sm text-destructive-foreground">Failed to load counterparties: {(error as Error).message}</div> : null}

      <div className="border border-border overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Focal Wallet</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Counterparty</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">First Seen</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Last Seen</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Inbound</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Outbound</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Last Outbound</th>
              <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Candidate Links</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {(data?.items ?? []).map((item) => (
              <tr key={`${item.focalWallet}-${item.counterpartyAddress}`}>
                <td className="px-4 py-3 text-sm font-mono">{shortAddress(item.focalWallet, 8, 6)}</td>
                <td className="px-4 py-3 text-sm font-mono">{shortAddress(item.counterpartyAddress, 8, 6)}</td>
                <td className="px-4 py-3 text-sm text-muted-foreground">{formatDateTime(item.firstSeenAt)}</td>
                <td className="px-4 py-3 text-sm text-muted-foreground">{formatDateTime(item.lastSeenAt)}</td>
                <td className="px-4 py-3 text-sm font-mono">{item.inboundCount}</td>
                <td className="px-4 py-3 text-sm font-mono">{item.outboundCount}</td>
                <td className="px-4 py-3 text-sm text-muted-foreground">{item.lastOutboundAt ? formatDateTime(item.lastOutboundAt) : "-"}</td>
                <td className="px-4 py-3 text-sm font-mono">{item.candidateLinks}</td>
              </tr>
            ))}
            {(data?.items?.length ?? 0) === 0 ? <tr><td className="px-4 py-4 text-sm text-muted-foreground" colSpan={8}>No counterparties found.</td></tr> : null}
          </tbody>
        </table>
      </div>

      <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
        <div className="font-mono">Showing page {page} • {data?.total ?? 0} total</div>
        <div className="flex gap-2">
          <button onClick={() => setPage(page - 1)} disabled={page <= 1} className="px-3 py-2 border border-border text-xs disabled:opacity-50">Previous</button>
          <button onClick={() => setPage(page + 1)} disabled={Boolean(data && page * pageSize >= data.total)} className="px-3 py-2 border border-border text-xs disabled:opacity-50">Next</button>
        </div>
      </div>
    </div>
  );
}
