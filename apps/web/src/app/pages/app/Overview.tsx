import { AlertTriangle, Clock, XCircle, Database } from "lucide-react";
import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import { getOverview, type OverviewResponse } from "../../lib/api";
import { formatDateTime, shortAddress, timeAgo } from "../../lib/format";

export default function Overview() {
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    void getOverview().then(setData).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load overview");
    });
  }, []);

  const metrics = data?.metrics;
  const candidates = data?.recentCandidates ?? [];

  return (
    <div className="p-8 max-w-7xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Overview</h1>
        <p className="text-muted-foreground text-sm">Scan window status and recent detection activity</p>
      </div>

      {error ? <div className="mb-8 text-sm text-destructive-foreground">{error}</div> : null}

      <div className="grid md:grid-cols-4 gap-12 mb-16 pb-16 border-b border-border">
        <Metric icon={<AlertTriangle className="w-4 h-4 text-muted-foreground" />} label="Candidates Emitted" value={String(metrics?.candidatesEmitted ?? 0)} sub="Last 24h" />
        <Metric icon={<XCircle className="w-4 h-4 text-muted-foreground" />} label="Unknown-Gate Blocks" value={String(metrics?.unknownGateBlocks ?? 0)} sub="Pending data availability" />
        <Metric icon={<Database className="w-4 h-4 text-muted-foreground" />} label="Transactions Scanned" value={String(metrics?.transactionsScanned ?? 0)} sub="Current scan window" />
        <Metric icon={<Clock className="w-4 h-4 text-muted-foreground" />} label="Last Scan Update" value={timeAgo(metrics?.lastScanUpdateAt ?? "")} sub={formatDateTime(metrics?.lastScanUpdateAt ?? "")} />
      </div>

      <div className="mb-16">
        <div className="flex items-center justify-between mb-8">
          <h2 className="text-lg tracking-tight">Recent Candidates</h2>
          <Link to="/app/candidates" className="text-xs text-muted-foreground hover:text-foreground transition-colors hover:underline underline-offset-4">
            View All →
          </Link>
        </div>
        <div className="border border-border">
          <div className="grid grid-cols-[1fr_auto_auto_auto] gap-6 px-6 py-4 border-b border-border bg-muted/30 text-xs uppercase tracking-widest text-muted-foreground font-mono">
            <div>Signature</div>
            <div>Counterparty</div>
            <div>Repeat Count</div>
            <div>Recency</div>
          </div>
          <div className="divide-y divide-border">
            {candidates.map((candidate, index) => (
              <Link key={`${candidate.signature}-${candidate.transferIndex}-${index}`} to="/app/candidates" className="grid grid-cols-[1fr_auto_auto_auto] gap-6 px-6 py-5 hover:bg-muted/30 transition-colors border-l-2 border-l-destructive-foreground">
                <div className="font-mono text-sm">{shortAddress(candidate.signature, 6, 6)} <span className="text-muted-foreground text-xs ml-2">{timeAgo(candidate.blockTime)}</span></div>
                <div className="font-mono text-sm">{shortAddress(candidate.suspiciousCounterparty, 6, 4)}</div>
                <div className="text-sm font-mono">{candidate.repeatInjectionCount}</div>
                <div className="text-sm text-muted-foreground">{candidate.recencyDays}d</div>
              </Link>
            ))}
            {candidates.length === 0 ? <div className="px-6 py-5 text-sm text-muted-foreground">No candidates yet.</div> : null}
          </div>
        </div>
      </div>

      <div className="border-t border-border pt-12">
        <h2 className="text-lg tracking-tight mb-8">Scan Window Summary (24h)</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-12">
          <Summary label="Candidates Emitted" value={String(metrics?.candidatesEmitted ?? 0)} />
          <Summary label="Passed Gates" value={String(metrics?.passedTransactions ?? 0)} />
          <Summary label="Unknown-Gate Blocked" value={String(metrics?.unknownGateBlocks ?? 0)} />
          <Summary label="Pass Rate" value={`${(metrics?.passRatePct ?? 0).toFixed(1)}%`} />
        </div>
      </div>
    </div>
  );
}

function Metric({ icon, label, value, sub }: { icon: JSX.Element; label: string; value: string; sub: string }) {
  return (
    <div>
      <div className="flex items-center gap-3 mb-3">
        {icon}
        <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono">{label}</div>
      </div>
      <div className="text-4xl font-mono tracking-tight">{value}</div>
      <div className="text-xs text-muted-foreground mt-2">{sub}</div>
    </div>
  );
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-3xl font-mono tracking-tight mb-2">{value}</div>
      <div className="text-xs text-muted-foreground uppercase tracking-widest">{label}</div>
    </div>
  );
}
