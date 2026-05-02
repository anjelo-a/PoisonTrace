import { Database, Settings as SettingsIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { getOverview, getSettings, getWalletSync, type SettingsResponse, type WalletSyncRow } from "../../lib/api";
import { timeAgo } from "../../lib/format";

export default function WalletSync() {
  const [settings, setSettings] = useState<SettingsResponse | null>(null);
  const [syncRows, setSyncRows] = useState<WalletSyncRow[]>([]);
  const [transactionsScanned, setTransactionsScanned] = useState(0);
  const [lastUpdate, setLastUpdate] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    void Promise.all([getSettings(), getWalletSync(1, 20), getOverview()])
      .then(([cfg, sync, overview]) => {
        setSettings(cfg);
        setSyncRows(sync.items);
        setTransactionsScanned(overview.metrics.transactionsScanned);
        setLastUpdate(overview.metrics.lastScanUpdateAt);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Failed to load scan configuration");
      });
  }, []);

  const latest = syncRows[0];
  const windowSummary = useMemo(() => {
    if (!latest) {
      return { baseline: "-", scan: "-" };
    }
    return {
      baseline: `${latest.baselineStartAt} → ${latest.baselineEndAt}`,
      scan: `${latest.scanStartAt} → ${latest.scanEndAt}`,
    };
  }, [latest]);

  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Scan Configuration</h1>
        <p className="text-muted-foreground text-sm">Current scan bounds, caps, and window definitions (read-only)</p>
      </div>
      {error ? <div className="mb-8 text-sm text-destructive-foreground">{error}</div> : null}

      <div className="border border-border mb-16">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <SettingsIcon className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Execution Bounds & Caps</div>
        </div>
        <div className="p-8 grid grid-cols-2 gap-x-12 gap-y-6 text-sm font-mono">
          <KV k="Max Wallets per Run" v={String(settings?.maxWalletsPerRun ?? "-")} />
          <KV k="Max TX Pages per Wallet" v={String(settings?.maxTXPagesPerWallet ?? "-")} />
          <KV k="Max TX per Wallet" v={String(settings?.maxTXPerWallet ?? "-")} />
          <KV k="Max Concurrent Wallets" v={String(settings?.maxConcurrentWallets ?? "-")} />
          <KV k="Wallet Timeout" v={`${settings?.walletSyncTimeoutSeconds ?? "-"}s`} />
          <KV k="Run Timeout" v={`${settings?.runTimeoutSeconds ?? "-"}s`} />
          <KV k="Max Retries" v={String(settings?.maxHeliusRetries ?? "-")} />
          <KV k="Request Delay" v={`${settings?.heliusRequestDelayMS ?? "-"}ms`} />
        </div>
      </div>

      <div className="border border-border mb-16">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Database className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Scan Window Definitions</div>
        </div>
        <div className="p-8 space-y-8">
          <div className="border border-border p-6 bg-muted/30 text-sm font-mono">
            <div className="text-xs text-muted-foreground mb-2">Baseline Window</div>
            <div>{windowSummary.baseline}</div>
          </div>
          <div className="border border-border p-6 bg-muted/30 text-sm font-mono">
            <div className="text-xs text-muted-foreground mb-2">Scan Window</div>
            <div>{windowSummary.scan}</div>
          </div>
        </div>
      </div>

      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 text-sm uppercase tracking-widest font-mono text-muted-foreground">Current Scan Status</div>
        <div className="p-8 grid grid-cols-3 gap-12">
          <RunStat label="Transactions Scanned" value={String(transactionsScanned)} />
          <RunStat label="Wallet Sync Rows Loaded" value={String(syncRows.length)} />
          <RunStat label="Last Update" value={timeAgo(lastUpdate)} />
        </div>
      </div>
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return <div className="flex justify-between py-3 border-b border-border"><span className="text-muted-foreground">{k}:</span><span>{v}</span></div>;
}

function RunStat({ label, value }: { label: string; value: string }) {
  return <div><div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">{label}</div><div className="text-2xl font-mono tracking-tight">{value}</div></div>;
}
