import { Shield, Filter, Download } from "lucide-react";

export default function Settings() {
  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Settings</h1>
        <p className="text-muted-foreground text-sm">Configure detection rules, filters, and export preferences</p>
      </div>

      {/* Detection Rules */}
      <div className="border border-border mb-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Shield className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Detection Rules</div>
        </div>
        <div className="p-8 space-y-8">
          <div>
            <label className="block text-sm mb-3">
              Flagging Threshold
            </label>
            <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
              <option>1 failed gate (most sensitive)</option>
              <option selected>2 failed gates (recommended)</option>
              <option>3 failed gates (less sensitive)</option>
              <option>4 failed gates (all gates must fail)</option>
            </select>
            <p className="text-xs text-muted-foreground mt-2 font-mono">
              Minimum number of failed gates required to flag a pattern as probable poisoning signal
            </p>
          </div>

          <div>
            <label className="block text-sm mb-3">
              Methodology Version
            </label>
            <div className="flex items-center justify-between px-4 py-3 border border-border bg-muted/30">
              <span className="text-sm text-muted-foreground font-mono">v1.0.0 (Current)</span>
              <span className="text-xs text-muted-foreground font-mono">May 2026</span>
            </div>
            <p className="text-xs text-muted-foreground mt-2 font-mono">
              Detection rules are automatically updated to latest version
            </p>
          </div>

          <div className="pt-6 border-t border-border">
            <button className="px-6 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors">
              Save Configuration
            </button>
          </div>
        </div>
      </div>

      {/* View Filters */}
      <div className="border border-border mb-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Filter className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">View Filters</div>
        </div>
        <div className="p-8 space-y-8">
          <div>
            <label className="block text-sm mb-3">
              Default Time Range
            </label>
            <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
              <option>Last 24h</option>
              <option selected>Last 7d</option>
              <option>Last 30d</option>
              <option>All Time</option>
            </select>
            <p className="text-xs text-muted-foreground mt-2 font-mono">
              Default time window for pattern review screens
            </p>
          </div>

          <div>
            <label className="block text-sm mb-3">
              Default Sort Order
            </label>
            <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
              <option selected>Latest First</option>
              <option>Oldest First</option>
              <option>Severity (High to Low)</option>
            </select>
            <p className="text-xs text-muted-foreground mt-2 font-mono">
              How flagged patterns are ordered in candidate list
            </p>
          </div>

          <div className="flex items-start gap-4">
            <input type="checkbox" id="show-passed" className="mt-1" />
            <div>
              <label htmlFor="show-passed" className="text-sm block mb-1">
                Include passed transactions in exports
              </label>
              <p className="text-xs text-muted-foreground font-mono">
                Export files will include transactions that passed all gates (not flagged)
              </p>
            </div>
          </div>

          <div className="pt-6 border-t border-border">
            <button className="px-6 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors">
              Save Preferences
            </button>
          </div>
        </div>
      </div>

      {/* Export Preferences */}
      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Download className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Export Preferences</div>
        </div>
        <div className="p-8 space-y-8">
          <div>
            <label className="block text-sm mb-3">
              Default Export Format
            </label>
            <select className="w-full px-4 py-3 bg-transparent border border-border text-muted-foreground hover:border-foreground focus:outline-none focus:border-foreground transition-colors">
              <option selected>PDF (Formatted Report)</option>
              <option>CSV (Data Export)</option>
              <option>JSON (Machine-Readable)</option>
            </select>
            <p className="text-xs text-muted-foreground mt-2 font-mono">
              Preferred format for evidence artifact exports
            </p>
          </div>

          <div className="flex items-start gap-4">
            <input type="checkbox" id="include-trace" className="mt-1" defaultChecked />
            <div>
              <label htmlFor="include-trace" className="text-sm block mb-1">
                Include full gate trace logs
              </label>
              <p className="text-xs text-muted-foreground font-mono">
                Exports include detailed gate evaluation logs for each transaction (increases file size)
              </p>
            </div>
          </div>

          <div className="flex items-start gap-4">
            <input type="checkbox" id="include-metadata" className="mt-1" defaultChecked />
            <div>
              <label htmlFor="include-metadata" className="text-sm block mb-1">
                Include detection metadata
              </label>
              <p className="text-xs text-muted-foreground font-mono">
                Exports include methodology version, scan window bounds, and rule configuration
              </p>
            </div>
          </div>

          <div className="pt-6 border-t border-border">
            <button className="px-6 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors">
              Save Preferences
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
