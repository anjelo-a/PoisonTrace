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
  const detailWalletSyncRunID = Number(params.get("wallet_sync_run_id") ?? "0");
  const detailSignature = params.get("signature") ?? "";
  const detailTransferIndex = Number(params.get("transfer_index") ?? "-1");

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

  const selectedForDetail = selected ?? (detailWalletSyncRunID > 0 && detailSignature && detailTransferIndex >= 0
    ? {
        walletSyncRunId: detailWalletSyncRunID,
        focalWallet: "",
        signature: detailSignature,
        transferIndex: detailTransferIndex,
        blockTime: "",
        suspiciousCounterparty: "",
        matchedLegitCounterparty: "",
        repeatInjectionCount: 0,
        recencyDays: 0,
      }
    : null);
  const hasDetailSelection = selectedForDetail !== null;

  const detailQuery = useQuery({
    queryKey: ["candidate-detail", selectedForDetail?.walletSyncRunId, selectedForDetail?.signature, selectedForDetail?.transferIndex],
      queryFn: () => apiClient.getCandidateExplanation(selectedForDetail!.walletSyncRunId, selectedForDetail!.signature, selectedForDetail!.transferIndex),
    enabled: hasDetailSelection,
  });

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    next.set(key, value);
    next.set("page", "1");
    setParams(next, { replace: false });
  };

  return (
    <div className="h-full flex">
      <div className={`flex-1 flex flex-col ${hasDetailSelection ? "hidden md:flex" : ""}`}>
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
                <tr key={`${candidate.walletSyncRunId}-${candidate.signature}-${candidate.transferIndex}`} onClick={() => {
                  setSelected(candidate);
                  const next = new URLSearchParams(params);
                  next.set("wallet_sync_run_id", String(candidate.walletSyncRunId));
                  next.set("signature", candidate.signature);
                  next.set("transfer_index", String(candidate.transferIndex));
                  setParams(next, { replace: false });
                }} className={`cursor-pointer hover:bg-muted/30 transition-colors ${selectedForDetail?.walletSyncRunId === candidate.walletSyncRunId && selectedForDetail?.signature === candidate.signature && selectedForDetail?.transferIndex === candidate.transferIndex ? "bg-muted/30" : ""}`}>
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

      {selectedForDetail ? (
        <div className="w-full md:w-[560px] border-l border-border flex flex-col">
          <div className="p-8 border-b border-border flex items-start justify-between">
            <div><div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Candidate Detail</div><div className="font-mono text-sm break-all">{selectedForDetail.signature}</div></div>
            <button onClick={() => {
              setSelected(null);
              const next = new URLSearchParams(params);
              next.delete("wallet_sync_run_id");
              next.delete("signature");
              next.delete("transfer_index");
              setParams(next, { replace: false });
            }} className="ml-6 p-2 hover:bg-muted/30"><X className="w-4 h-4 text-muted-foreground" /></button>
          </div>
          <div className="flex-1 overflow-auto p-8 space-y-6 text-sm font-mono">
            {detailQuery.isLoading ? <div className="text-muted-foreground text-sm">Loading evidence...</div> : null}
            {detailQuery.isError ? <div className="text-destructive-foreground text-sm">Failed to load evidence detail.</div> : null}
            {detailQuery.data ? (
              <>
                <Row label="Run ID" value={String(detailQuery.data.runId)} />
                <Row label="Wallet Sync Run ID" value={String(detailQuery.data.walletSyncRunId)} />
                <Row label="Focal Wallet" value={detailQuery.data.focalWallet} mono />
                <Row label="Block Time" value={formatDateTime(detailQuery.data.blockTime)} />
                <Row label="Suspicious Counterparty" value={detailQuery.data.suspiciousCounterparty} mono />
                <Row label="Matched Legit" value={detailQuery.data.matchedLegitCounterparty} danger mono />
                <Row label="Relation Type" value={detailQuery.data.relationType} />
                <Row label="Asset Type" value={detailQuery.data.assetType} />
                <Row label="Normalization Status" value={detailQuery.data.normalizationStatus} />
                <Row label="Poisoning Eligible" value={String(detailQuery.data.poisoningEligible)} />
                <Row label="Source Owner" value={detailQuery.data.sourceOwner} mono />
                <Row label="Destination Owner" value={detailQuery.data.destinationOwner} mono />
                <Row label="From Token Account" value={detailQuery.data.fromTokenAccount} mono />
                <Row label="To Token Account" value={detailQuery.data.toTokenAccount} mono />
                <Row label="Token Mint" value={detailQuery.data.tokenMint} mono />
                <Row label="Amount Raw" value={detailQuery.data.amountRaw} mono />
                <Row label="Dust Status" value={detailQuery.data.dustStatus} />
                <Row label="Is Dust" value={String(detailQuery.data.isDust)} />
                <Row label="Is Zero Value" value={String(detailQuery.data.isZeroValue)} />
                <Row label="Is Inbound" value={String(detailQuery.data.isInbound)} />
                <Row label="Is New Counterparty" value={String(detailQuery.data.isNewCounterparty)} />
                <Row label="Recency Days" value={String(detailQuery.data.recencyDays)} />
                <Row label="Repeat Injections" value={`${detailQuery.data.repeatInjectionCount} events`} />
                <Row label="Lookalike Prefix Match" value={String(detailQuery.data.lookalikePrefixMatch)} />
                <Row label="Lookalike Suffix Match" value={String(detailQuery.data.lookalikeSuffixMatch)} />
                <Row label="Match Rule Version" value={detailQuery.data.matchRuleVersion} />
                <Row label="Legit Last Seen At" value={formatDateTime(detailQuery.data.legitLastSeenAt)} />
                <Row label="Baseline Complete" value={String(detailQuery.data.baselineComplete)} />
                <Row label="Incomplete Window" value={String(detailQuery.data.incompleteWindow)} />
                <Row label="Unknown Gate Reason" value={detailQuery.data.unknownGateReason} />
                <Row label="Scan Start At" value={formatDateTime(detailQuery.data.scanStartAt)} />
                <Row label="Scan End At" value={formatDateTime(detailQuery.data.scanEndAt)} />
                <Row label="Baseline Start At" value={formatDateTime(detailQuery.data.baselineStartAt)} />
                <Row label="Baseline End At" value={formatDateTime(detailQuery.data.baselineEndAt)} />
                <Row label="Source Ref Wallet Sync Run ID" value={String(detailQuery.data.sourceReferences?.walletSyncRunId ?? "")} />
                <Row label="Source Ref Run ID" value={String(detailQuery.data.sourceReferences?.runId ?? "")} />
                <Row label="Source Ref Transaction ID" value={String(detailQuery.data.sourceReferences?.transactionId ?? "")} />
                <Row label="Source Ref Wallet Transaction ID" value={String(detailQuery.data.sourceReferences?.walletTransactionId ?? "")} />
                <Row label="Source Ref Counterparty ID" value={String(detailQuery.data.sourceReferences?.counterpartyId ?? "")} />
              </>
            ) : null}
            <div className="text-xs text-muted-foreground">This view shows probable candidate evidence only. Unknown-gate blocked events are excluded from emitted candidates by contract.</div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function displayValue(value: string): string {
  if (value === "") return "(empty)";
  if (value === "unknown") return "unknown";
  return value;
}

function Row({ label, value, danger = false, mono = false }: { label: string; value: string; danger?: boolean; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[220px_1fr] gap-2">
      <span className="text-muted-foreground">{label}:</span>
      <span className={`${danger ? "text-destructive-foreground" : ""} ${mono ? "break-all" : ""}`}>{displayValue(value)}</span>
    </div>
  );
}
