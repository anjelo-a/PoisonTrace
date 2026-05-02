import { Download, FileText } from "lucide-react";
import { useEffect, useState } from "react";
import { getExports, type ExportRow } from "../../lib/api";
import { formatDateTime, timeAgo } from "../../lib/format";

export default function Exports() {
  const [exports, setExports] = useState<ExportRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void getExports().then((res) => setExports(res.items)).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load exports");
    });
  }, []);

  return (
    <div className="p-8 max-w-6xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Exports</h1>
        <p className="text-muted-foreground text-sm">Dataset export history</p>
      </div>

      {error ? <div className="mb-8 text-sm text-destructive-foreground">{error}</div> : null}

      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 text-sm uppercase tracking-widest font-mono text-muted-foreground">Export History</div>
        <div className="divide-y divide-border">
          {exports.map((exp) => (
            <div key={exp.id} className="p-8 flex items-center justify-between hover:bg-muted/30 transition-colors">
              <div className="flex items-center gap-6">
                <FileText className="w-6 h-6 text-muted-foreground" />
                <div>
                  <div className="text-sm mb-2">{exp.type}</div>
                  <div className="text-xs text-muted-foreground font-mono">{formatDateTime(exp.timestamp)} • {exp.format} • run-{exp.runId}</div>
                </div>
              </div>
              <div className="flex items-center gap-6">
                <div className="text-sm text-muted-foreground">{timeAgo(exp.timestamp)}</div>
                <button className="flex items-center gap-3 px-6 py-3 border border-border text-muted-foreground cursor-not-allowed">
                  <Download className="w-4 h-4" />
                  <span className="text-sm">Pending Artifact Link</span>
                </button>
              </div>
            </div>
          ))}
          {exports.length === 0 ? <div className="p-8 text-sm text-muted-foreground">No exports available yet.</div> : null}
        </div>
      </div>
    </div>
  );
}
