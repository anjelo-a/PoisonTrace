import { CheckCircle, XCircle, Clock } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { getRuns, type RunRow } from "../../lib/api";
import { percentFromString, timeAgo } from "../../lib/format";

export default function Runs() {
  const [runs, setRuns] = useState<RunRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void getRuns(1, 100).then((res) => setRuns(res.items)).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load runs");
    });
  }, []);

  const stats = useMemo(() => {
    const total = runs.length;
    const completed = runs.filter((r) => r.status === "completed").length;
    const partialOrTimeout = runs.filter((r) => r.status !== "completed").length;
    const unknown = runs.filter((r) => r.unknownGatePresent).length;
    return { total, completed, partialOrTimeout, unknown };
  }, [runs]);

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Detection Runs</h1>
        <p className="text-muted-foreground text-sm">Historical run logs with fail-closed and bounded execution outcomes</p>
      </div>
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">{error}</div> : null}
      <div className="px-8 py-6 border-b border-border grid grid-cols-4 gap-12">
        <RunStat label="Total Runs" value={String(stats.total)} />
        <RunStat label="Completed" value={String(stats.completed)} />
        <RunStat label="Partial/Failed" value={String(stats.partialOrTimeout)} />
        <RunStat label="Unknown Gates" value={String(stats.unknown)} />
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Run</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Status</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Wallets P/F/S</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Incomplete</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Truncation Rate</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Unknown Gates</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {runs.map((run) => (
              <tr key={run.id} className="hover:bg-muted/30 transition-colors">
                <td className="px-8 py-5"><div className="font-mono text-sm">run-{run.id}</div><div className="text-xs text-muted-foreground mt-1">{timeAgo(run.startedAt)}</div></td>
                <td className="px-8 py-5">{run.status === "completed" ? <div className="flex items-center gap-2"><CheckCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm">Completed</span></div> : run.status === "partial" ? <div className="flex items-center gap-2"><XCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm underline underline-offset-4">Partial</span></div> : <div className="flex items-center gap-2"><Clock className="w-4 h-4 text-muted-foreground" /><span className="text-sm text-destructive-foreground">{run.status}</span></div>}</td>
                <td className="px-8 py-5 text-sm font-mono">{run.walletsProcessed}/{run.walletsFailed}/{run.walletsSkipped}</td>
                <td className="px-8 py-5 text-sm font-mono">{run.incompleteWindows}</td>
                <td className="px-8 py-5 text-sm font-mono text-muted-foreground">{percentFromString(run.truncationWalletRate)}</td>
                <td className="px-8 py-5 text-sm">{run.unknownGatePresent ? "Yes" : "No"}</td>
              </tr>
            ))}
            {runs.length === 0 ? <tr><td className="px-8 py-5 text-sm text-muted-foreground" colSpan={6}>No runs found.</td></tr> : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function RunStat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">{label}</div>
      <div className="text-2xl font-mono tracking-tight">{value}</div>
    </div>
  );
}
