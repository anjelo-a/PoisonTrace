import { useEffect, useState } from "react";
import { getCandidates, type CandidateRow } from "../../lib/api";
import { formatDateTime, shortAddress, timeAgo } from "../../lib/format";

export default function Candidates() {
  const [items, setItems] = useState<CandidateRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void getCandidates(1, 100).then((res) => setItems(res.items)).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load candidates");
    });
  }, []);

  return (
    <div className="h-full flex flex-col">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Candidates</h1>
        <p className="text-muted-foreground text-sm">Emitted candidates requiring review</p>
      </div>
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">{error}</div> : null}
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
            {items.map((candidate) => (
              <tr key={`${candidate.signature}-${candidate.transferIndex}`} className="hover:bg-muted/30 transition-colors border-l-2 border-l-destructive-foreground">
                <td className="px-8 py-5 font-mono text-sm">{shortAddress(candidate.signature, 6, 6)}<div className="text-xs text-muted-foreground mt-1">{timeAgo(candidate.blockTime)}</div></td>
                <td className="px-8 py-5 text-sm text-muted-foreground font-mono">{formatDateTime(candidate.blockTime)}</td>
                <td className="px-8 py-5 font-mono text-sm">{shortAddress(candidate.suspiciousCounterparty, 6, 4)}</td>
                <td className="px-8 py-5 font-mono text-sm text-destructive-foreground">{shortAddress(candidate.matchedLegitCounterparty, 6, 4)}</td>
                <td className="px-8 py-5 text-sm font-mono">{candidate.repeatInjectionCount}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{candidate.recencyDays}d</td>
              </tr>
            ))}
            {items.length === 0 ? <tr><td className="px-8 py-5 text-sm text-muted-foreground" colSpan={6}>No candidates found.</td></tr> : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
