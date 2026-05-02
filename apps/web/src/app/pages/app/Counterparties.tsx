import { AlertTriangle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { getCounterparties, type CounterpartyRow } from "../../lib/api";
import { shortAddress, timeAgo } from "../../lib/format";

export default function Counterparties() {
  const [counterparties, setCounterparties] = useState<CounterpartyRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void getCounterparties(1, 100).then((res) => setCounterparties(res.items)).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load counterparties");
    });
  }, []);

  const stats = useMemo(() => {
    const total = counterparties.length;
    const inboundOnly = counterparties.filter((cp) => cp.outboundCount === 0).length;
    const withLinks = counterparties.filter((cp) => cp.candidateLinks > 0).length;
    const new24h = counterparties.filter((cp) => Date.now() - new Date(cp.firstSeenAt).getTime() <= 24 * 60 * 60 * 1000).length;
    return { total, inboundOnly, withLinks, new24h };
  }, [counterparties]);

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Counterparties</h1>
        <p className="text-muted-foreground text-sm">Address interaction history with deterministic metadata</p>
      </div>
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">{error}</div> : null}

      <div className="px-8 py-6 border-b border-border grid grid-cols-4 gap-12">
        <Stat label="Total Counterparties" value={String(stats.total)} />
        <Stat label="Inbound Only" value={String(stats.inboundOnly)} />
        <Stat label="With Candidate Links" value={String(stats.withLinks)} />
        <Stat label="New (24h)" value={String(stats.new24h)} />
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Address</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">First Seen</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Last Seen</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Inbound</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Outbound</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Last Outbound</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Candidate Links</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {counterparties.map((cp) => (
              <tr key={`${cp.focalWallet}-${cp.counterpartyAddress}`} className={`hover:bg-muted/30 transition-colors ${cp.candidateLinks > 0 ? "border-l-2 border-l-destructive-foreground" : ""}`}>
                <td className="px-8 py-5 font-mono text-sm">{shortAddress(cp.counterpartyAddress, 6, 4)}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(cp.firstSeenAt)}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(cp.lastSeenAt)}</td>
                <td className="px-8 py-5 text-sm font-mono">{cp.inboundCount}</td>
                <td className="px-8 py-5 text-sm font-mono text-muted-foreground">{cp.outboundCount}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{cp.lastOutboundAt ? timeAgo(cp.lastOutboundAt) : "—"}</td>
                <td className="px-8 py-5">{cp.candidateLinks > 0 ? <div className="flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-muted-foreground" /><span className="text-sm">{cp.candidateLinks}</span></div> : <span className="text-sm text-muted-foreground">0</span>}</td>
              </tr>
            ))}
            {counterparties.length === 0 ? <tr><td className="px-8 py-5 text-sm text-muted-foreground" colSpan={7}>No counterparties found.</td></tr> : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div><div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">{label}</div><div className="text-2xl font-mono tracking-tight">{value}</div></div>;
}
