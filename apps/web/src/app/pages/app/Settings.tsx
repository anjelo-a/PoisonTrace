import { useQuery } from "@tanstack/react-query";
import { Shield, Timer, Search, Database } from "lucide-react";
import { apiClient } from "../../lib/apiClient";

export default function Settings() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiClient.getSettings(),
  });

  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Settings</h1>
        <p className="text-muted-foreground text-sm">Read-only backend configuration from `GET /api/settings`</p>
      </div>

      {isLoading ? <div className="mb-8 text-sm text-muted-foreground">Loading backend settings...</div> : null}
      {error ? <div className="mb-8 text-sm text-destructive-foreground">Failed to load settings: {(error as Error).message}</div> : null}

      <div className="mb-8 border border-border bg-muted/20 px-4 py-3 text-xs text-muted-foreground">
        Settings are read-only in this phase. Write/edit actions are intentionally disabled because no settings write endpoint exists yet.
      </div>

      <div className="border border-border mb-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Shield className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Runtime Bounds</div>
        </div>
        <div className="p-8 grid md:grid-cols-2 gap-6 text-sm font-mono">
          <KV k="Max Wallets per Run" v={String(data?.maxWalletsPerRun ?? "-")} />
          <KV k="Max TX Pages per Wallet" v={String(data?.maxTXPagesPerWallet ?? "-")} />
          <KV k="Max TX per Wallet" v={String(data?.maxTXPerWallet ?? "-")} />
          <KV k="Max Concurrent Wallets" v={String(data?.maxConcurrentWallets ?? "-")} />
          <KV k="Max Helius Retries" v={String(data?.maxHeliusRetries ?? "-")} />
          <KV k="Helius Request Delay MS" v={String(data?.heliusRequestDelayMS ?? "-")} />
        </div>
      </div>

      <div className="border border-border mb-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Timer className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Timeouts and Windows</div>
        </div>
        <div className="p-8 grid md:grid-cols-2 gap-6 text-sm font-mono">
          <KV k="Wallet Sync Timeout Seconds" v={String(data?.walletSyncTimeoutSeconds ?? "-")} />
          <KV k="Run Timeout Seconds" v={String(data?.runTimeoutSeconds ?? "-")} />
          <KV k="Baseline Lookback Days" v={String(data?.baselineLookbackDays ?? "-")} />
          <KV k="Scan Window Days" v={String(data?.scanWindowDays ?? "-")} />
        </div>
      </div>

      <div className="border border-border">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Search className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Detection Gates</div>
        </div>
        <div className="p-8 grid md:grid-cols-2 gap-6 text-sm font-mono">
          <KV k="Lookalike Recency Days" v={String(data?.lookalikeRecencyDays ?? "-")} />
          <KV k="Lookalike Prefix Min" v={String(data?.lookalikePrefixMin ?? "-")} />
          <KV k="Lookalike Suffix Min" v={String(data?.lookalikeSuffixMin ?? "-")} />
          <KV k="Lookalike Single-Side Min" v={String(data?.lookalikeSingleSideMin ?? "-")} />
          <KV k="Minimum Injection Count" v={String(data?.minInjectionCount ?? "-")} />
        </div>
      </div>

      <div className="border border-border mt-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Database className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Mutability Status</div>
        </div>
        <div className="p-8 text-sm text-muted-foreground">
          Backend config is currently API read-only.
          <br />
          Write endpoint status: <span className="font-mono">not yet API-backed</span>
        </div>
      </div>
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return <div className="flex justify-between py-3 border-b border-border"><span className="text-muted-foreground">{k}:</span><span>{v}</span></div>;
}
