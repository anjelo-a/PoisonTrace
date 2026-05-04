import { Download, FileText } from "lucide-react";

export default function Exports() {
  const exports = [
    {
      id: "export-2026-05-01-001",
      timestamp: "2026-05-01 14:15:32",
      timeAgo: "19m ago",
      type: "Candidate Report",
      records: 7,
      format: "PDF",
      size: "342 KB",
      status: "Ready",
    },
    {
      id: "export-2026-04-30-002",
      timestamp: "2026-04-30 18:22:15",
      timeAgo: "20h ago",
      type: "Transaction Log",
      records: 1243,
      format: "CSV",
      size: "1.2 MB",
      status: "Ready",
    },
    {
      id: "export-2026-04-29-001",
      timestamp: "2026-04-29 12:10:08",
      timeAgo: "2d ago",
      type: "Full Audit",
      records: 3847,
      format: "JSON",
      size: "4.5 MB",
      status: "Ready",
    },
  ];

  return (
    <div className="p-8 max-w-6xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Exports</h1>
        <p className="text-muted-foreground text-sm">Generate and download forensic reports</p>
      </div>

      {/* Create New Export */}
      <div className="border border-border mb-16">
        <div className="px-8 py-4 border-b border-border bg-muted/30 text-sm uppercase tracking-widest font-mono text-muted-foreground">
          Create New Export
        </div>
        <div className="p-8 space-y-8">
          <div className="grid md:grid-cols-2 gap-8">
            <div>
              <label className="block text-sm mb-3">
                Export Type
              </label>
              <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
                <option>Candidate Report</option>
                <option>Transaction Log</option>
                <option>Detection Run Summary</option>
                <option>Counterparty Analysis</option>
                <option>Full Audit Report</option>
              </select>
            </div>

            <div>
              <label className="block text-sm mb-3">
                Format
              </label>
              <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
                <option>PDF (Formatted Report)</option>
                <option>CSV (Data Export)</option>
                <option>JSON (Machine-Readable)</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm mb-3">
              Date Range
            </label>
            <div className="grid md:grid-cols-2 gap-6">
              <input
                type="date"
                className="px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors"
                defaultValue="2026-04-01"
              />
              <input
                type="date"
                className="px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors"
                defaultValue="2026-05-01"
              />
            </div>
          </div>

          <div className="flex items-start gap-4">
            <input type="checkbox" id="include-passed" className="mt-1" />
            <div>
              <label htmlFor="include-passed" className="text-sm block mb-1">
                Include passed transactions
              </label>
              <p className="text-xs text-muted-foreground font-mono">
                Export includes transactions that passed all gates (not flagged)
              </p>
            </div>
          </div>

          <div className="flex items-start gap-4">
            <input type="checkbox" id="include-trace" className="mt-1" defaultChecked />
            <div>
              <label htmlFor="include-trace" className="text-sm block mb-1">
                Include full gate trace logs
              </label>
              <p className="text-xs text-muted-foreground font-mono">
                Detailed gate evaluation logs for each transaction (increases file size)
              </p>
            </div>
          </div>

          <div className="pt-6 border-t border-border">
            <button className="flex items-center gap-3 px-6 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors">
              <Download className="w-4 h-4" />
              <span>Generate Export</span>
            </button>
          </div>
        </div>
      </div>

      {/* Export History */}
      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 text-sm uppercase tracking-widest font-mono text-muted-foreground">
          Export History
        </div>
        <div className="divide-y divide-border">
          {exports.map((exp) => (
            <div key={exp.id} className="p-8 flex items-center justify-between hover:bg-muted/30 transition-colors">
              <div className="flex items-center gap-6">
                <FileText className="w-6 h-6 text-muted-foreground" />
                <div>
                  <div className="text-sm mb-2">{exp.type}</div>
                  <div className="text-xs text-muted-foreground font-mono">
                    {exp.timestamp} • {exp.records} records • {exp.format} • {exp.size}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-6">
                <div className="text-sm text-muted-foreground">{exp.timeAgo}</div>
                <button className="flex items-center gap-3 px-6 py-3 border border-border hover:border-foreground hover:text-foreground transition-colors">
                  <Download className="w-4 h-4" />
                  <span className="text-sm">Download</span>
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
