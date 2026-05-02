import { Database, Settings as SettingsIcon } from "lucide-react";

export default function WalletSync() {
  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Scan Configuration</h1>
        <p className="text-muted-foreground text-sm">Current scan bounds, caps, and window definitions (read-only)</p>
      </div>

      {/* Execution Bounds & Caps */}
      <div className="border border-border mb-16">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <SettingsIcon className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Execution Bounds & Caps</div>
        </div>
        <div className="p-8">
          <div className="grid grid-cols-2 gap-x-12 gap-y-6 text-sm font-mono">
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Max Wallets per Run:</span>
              <span>50</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Max TX Pages per Wallet:</span>
              <span>10</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Max TX per Wallet:</span>
              <span>1000</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Max Concurrent Wallets:</span>
              <span>5</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Wallet Timeout:</span>
              <span>30s</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Run Timeout:</span>
              <span>300s</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Max Retries:</span>
              <span>3</span>
            </div>
            <div className="flex justify-between py-3 border-b border-border">
              <span className="text-muted-foreground">Request Delay:</span>
              <span>100ms</span>
            </div>
          </div>
        </div>
      </div>

      {/* Scan Window Definitions */}
      <div className="border border-border mb-16">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Database className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Scan Window Definitions</div>
        </div>
        <div className="p-8 space-y-8">
          <div>
            <div className="text-sm mb-4">Baseline Window</div>
            <div className="border border-border p-6 bg-muted/30">
              <div className="grid grid-cols-2 gap-6 text-sm font-mono">
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Purpose:</div>
                  <div>Establish known counterparties</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Default Depth:</div>
                  <div>90 days</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Status:</div>
                  <div>Active</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Last Update:</div>
                  <div>2m ago</div>
                </div>
              </div>
            </div>
          </div>

          <div>
            <div className="text-sm mb-4">Scan Window</div>
            <div className="border border-border p-6 bg-muted/30">
              <div className="grid grid-cols-2 gap-6 text-sm font-mono">
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Purpose:</div>
                  <div>Analyze recent activity for patterns</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Default Depth:</div>
                  <div>7 days</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Status:</div>
                  <div>Active</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-2">Last Update:</div>
                  <div>2m ago</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Current Scan Status */}
      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 text-sm uppercase tracking-widest font-mono text-muted-foreground">
          Current Scan Status
        </div>
        <div className="p-8">
          <div className="grid grid-cols-3 gap-12">
            <div>
              <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Transactions Scanned</div>
              <div className="text-2xl font-mono tracking-tight">1,243</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Last Block</div>
              <div className="text-2xl font-mono tracking-tight">285,493,721</div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Last Update</div>
              <div className="text-2xl font-mono tracking-tight">2m ago</div>
            </div>
          </div>

          <p className="text-sm text-muted-foreground mt-8 leading-relaxed">
            Configuration values are set at system level. Contact support for bounded window adjustments.
          </p>
        </div>
      </div>
    </div>
  );
}
