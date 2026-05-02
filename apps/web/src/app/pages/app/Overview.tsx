import { AlertTriangle, Clock, XCircle, Database } from "lucide-react";
import { Link } from "react-router-dom";

export default function Overview() {
  return (
    <div className="p-8 max-w-7xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Overview</h1>
        <p className="text-muted-foreground text-sm">Scan window status and recent detection activity</p>
      </div>

      {/* Status Metrics */}
      <div className="grid md:grid-cols-4 gap-12 mb-16 pb-16 border-b border-border">
        <div>
          <div className="flex items-center gap-3 mb-3">
            <AlertTriangle className="w-4 h-4 text-muted-foreground" />
            <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono">Candidates Emitted</div>
          </div>
          <div className="text-4xl font-mono tracking-tight">7</div>
          <div className="text-xs text-muted-foreground mt-2">Last 24h</div>
        </div>

        <div>
          <div className="flex items-center gap-3 mb-3">
            <XCircle className="w-4 h-4 text-muted-foreground" />
            <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono">Unknown-Gate Blocks</div>
          </div>
          <div className="text-4xl font-mono tracking-tight">3</div>
          <div className="text-xs text-muted-foreground mt-2">Pending data availability</div>
        </div>

        <div>
          <div className="flex items-center gap-3 mb-3">
            <Database className="w-4 h-4 text-muted-foreground" />
            <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono">Transactions Scanned</div>
          </div>
          <div className="text-4xl font-mono tracking-tight">1,243</div>
          <div className="text-xs text-muted-foreground mt-2">Current scan window</div>
        </div>

        <div>
          <div className="flex items-center gap-3 mb-3">
            <Clock className="w-4 h-4 text-muted-foreground" />
            <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono">Last Scan Update</div>
          </div>
          <div className="text-4xl font-mono tracking-tight">2m</div>
          <div className="text-xs text-muted-foreground mt-2">ago</div>
        </div>
      </div>

      {/* Recent Candidates */}
      <div className="mb-16">
        <div className="flex items-center justify-between mb-8">
          <h2 className="text-lg tracking-tight">Recent Candidates</h2>
          <Link to="/app/candidates" className="text-xs text-muted-foreground hover:text-foreground transition-colors hover:underline underline-offset-4">
            View All →
          </Link>
        </div>
        <div className="border border-border">
          <div className="grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-6 px-6 py-4 border-b border-border bg-muted/30 text-xs uppercase tracking-widest text-muted-foreground font-mono">
            <div>Signature</div>
            <div>Block Time</div>
            <div>Suspicious Counterparty</div>
            <div>Repeat Count</div>
            <div>Recency</div>
            <div>Status</div>
          </div>
          <div className="divide-y divide-border">
            {[
              { sig: "5KqR...d8Hs", time: "2m ago", counterparty: "9xyz...4abc", repeats: 3, recency: "18h", status: "Needs Review" },
              { sig: "9Lmn...x3Ty", time: "5m ago", counterparty: "2def...8xyz", repeats: 5, recency: "6h", status: "Needs Review" },
              { sig: "1Abc...p7Qw", time: "12m ago", counterparty: "7mno...1pqr", repeats: 2, recency: "22h", status: "Needs Review" },
              { sig: "7Zxy...k4Mn", time: "18m ago", counterparty: "5stu...6vwx", repeats: 4, recency: "12h", status: "Needs Review" },
              { sig: "3Def...r9Ij", time: "25m ago", counterparty: "4yza...3bcd", repeats: 6, recency: "4h", status: "Needs Review" },
            ].map((candidate) => (
              <Link
                key={candidate.sig}
                to="/app/candidates"
                className="grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-6 px-6 py-5 hover:bg-muted/30 transition-colors border-l-2 border-l-destructive-foreground"
              >
                <div className="font-mono text-sm">{candidate.sig}</div>
                <div className="text-sm text-muted-foreground w-20">{candidate.time}</div>
                <div className="font-mono text-sm w-32">{candidate.counterparty}</div>
                <div className="text-sm font-mono w-24">{candidate.repeats} events</div>
                <div className="text-sm text-muted-foreground w-16">{candidate.recency}</div>
                <div className="text-xs font-mono w-32 underline underline-offset-4">
                  {candidate.status}
                </div>
              </Link>
            ))}
          </div>
        </div>
      </div>

      {/* Detection Summary */}
      <div className="border-t border-border pt-12">
        <h2 className="text-lg tracking-tight mb-8">Scan Window Summary (24h)</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-12">
          <div>
            <div className="text-3xl font-mono tracking-tight mb-2">7</div>
            <div className="text-xs text-muted-foreground uppercase tracking-widest">Candidates Emitted</div>
          </div>
          <div>
            <div className="text-3xl font-mono tracking-tight mb-2">1,236</div>
            <div className="text-xs text-muted-foreground uppercase tracking-widest">Passed Gates</div>
          </div>
          <div>
            <div className="text-3xl font-mono tracking-tight mb-2">3</div>
            <div className="text-xs text-muted-foreground uppercase tracking-widest">Unknown-Gate Blocked</div>
          </div>
          <div>
            <div className="text-3xl font-mono tracking-tight mb-2">99.4%</div>
            <div className="text-xs text-muted-foreground uppercase tracking-widest">Pass Rate</div>
          </div>
        </div>
      </div>
    </div>
  );
}
