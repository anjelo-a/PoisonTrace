import { CheckCircle, XCircle, Clock } from "lucide-react";

export default function Runs() {
  const runs = [
    {
      id: "run-2026-05-01-143",
      timestamp: "2026-05-01 14:32:45",
      timeAgo: "2m ago",
      status: "Completed",
      walletsProcessed: 12,
      walletsFailed: 0,
      walletsSkipped: 0,
      incompleteWindows: 0,
      truncationRate: "0%",
      unknownGatePresent: false,
      retryExhausted: 0,
      duration: "1.2s",
    },
    {
      id: "run-2026-05-01-142",
      timestamp: "2026-05-01 14:27:30",
      timeAgo: "7m ago",
      status: "Completed",
      walletsProcessed: 15,
      walletsFailed: 1,
      walletsSkipped: 0,
      incompleteWindows: 1,
      truncationRate: "6.7%",
      unknownGatePresent: true,
      retryExhausted: 0,
      duration: "1.8s",
    },
    {
      id: "run-2026-05-01-141",
      timestamp: "2026-05-01 14:22:15",
      timeAgo: "12m ago",
      status: "Completed",
      walletsProcessed: 10,
      walletsFailed: 0,
      walletsSkipped: 0,
      incompleteWindows: 0,
      truncationRate: "0%",
      unknownGatePresent: false,
      retryExhausted: 0,
      duration: "1.1s",
    },
    {
      id: "run-2026-05-01-140",
      timestamp: "2026-05-01 14:17:02",
      timeAgo: "17m ago",
      status: "Partial",
      walletsProcessed: 8,
      walletsFailed: 2,
      walletsSkipped: 1,
      incompleteWindows: 2,
      truncationRate: "25%",
      unknownGatePresent: true,
      retryExhausted: 1,
      duration: "2.1s",
    },
    {
      id: "run-2026-05-01-139",
      timestamp: "2026-05-01 14:11:48",
      timeAgo: "23m ago",
      status: "Timed Out",
      walletsProcessed: 5,
      walletsFailed: 3,
      walletsSkipped: 2,
      incompleteWindows: 3,
      truncationRate: "60%",
      unknownGatePresent: true,
      retryExhausted: 2,
      duration: "10.0s",
    },
  ];

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Detection Runs</h1>
        <p className="text-muted-foreground text-sm">Historical run logs with fail-closed and bounded execution outcomes</p>
      </div>

      {/* Stats */}
      <div className="px-8 py-6 border-b border-border grid grid-cols-5 gap-12">
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Total Runs (24h)</div>
          <div className="text-2xl font-mono tracking-tight">287</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Avg Duration</div>
          <div className="text-2xl font-mono tracking-tight">1.4s</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Completed</div>
          <div className="text-2xl font-mono tracking-tight">285</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Partial/Timeout</div>
          <div className="text-2xl font-mono tracking-tight">2</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Unknown Gates</div>
          <div className="text-2xl font-mono tracking-tight">3</div>
        </div>
      </div>

      {/* Run List */}
      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Run ID
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Timestamp
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Status
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Wallets P/F/S
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Incomplete Windows
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Truncation Rate
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Unknown Gates
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Duration
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {runs.map((run) => (
              <tr key={run.id} className="hover:bg-muted/30 transition-colors">
                <td className="px-8 py-5">
                  <div className="font-mono text-sm">{run.id}</div>
                  <div className="text-xs text-muted-foreground mt-1">{run.timeAgo}</div>
                </td>
                <td className="px-8 py-5 text-sm font-mono">{run.timestamp}</td>
                <td className="px-8 py-5">
                  {run.status === "Completed" ? (
                    <div className="flex items-center gap-2">
                      <CheckCircle className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm">Completed</span>
                    </div>
                  ) : run.status === "Partial" ? (
                    <div className="flex items-center gap-2">
                      <XCircle className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm underline underline-offset-4">Partial</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <Clock className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm text-destructive-foreground">Timed Out</span>
                    </div>
                  )}
                </td>
                <td className="px-8 py-5 text-sm font-mono">
                  {run.walletsProcessed}/{run.walletsFailed}/{run.walletsSkipped}
                </td>
                <td className="px-8 py-5 text-sm font-mono">{run.incompleteWindows}</td>
                <td className="px-8 py-5 text-sm font-mono text-muted-foreground">{run.truncationRate}</td>
                <td className="px-8 py-5">
                  {run.unknownGatePresent ? (
                    <span className="text-sm underline underline-offset-4">Yes</span>
                  ) : (
                    <span className="text-sm text-muted-foreground">No</span>
                  )}
                </td>
                <td className="px-8 py-5 text-sm font-mono text-muted-foreground">{run.duration}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground">
        <div className="font-mono">Showing 5 of 287 runs</div>
        <div className="flex gap-3">
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">Previous</button>
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">Next</button>
        </div>
      </div>
    </div>
  );
}
