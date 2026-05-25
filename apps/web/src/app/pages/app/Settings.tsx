import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Database, Save, Search, Shield, Timer } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { apiClient, type SettingsResponse } from "../../lib/apiClient";

const settingGroups: Array<{
  title: string;
  icon: typeof Shield;
  fields: Array<{ key: keyof SettingsResponse; label: string; min: number }>;
}> = [
  {
    title: "Runtime Bounds",
    icon: Shield,
    fields: [
      { key: "maxWalletsPerRun", label: "Max Wallets per Run", min: 1 },
      { key: "maxTXPagesPerWallet", label: "Max TX Pages per Wallet", min: 1 },
      { key: "maxTXPerWallet", label: "Max TX per Wallet", min: 1 },
      { key: "maxConcurrentWallets", label: "Max Concurrent Wallets", min: 1 },
      { key: "maxHeliusRetries", label: "Max Helius Retries", min: 0 },
      { key: "heliusRequestDelayMS", label: "Helius Request Delay MS", min: 0 },
    ],
  },
  {
    title: "Timeouts and Windows",
    icon: Timer,
    fields: [
      { key: "walletSyncTimeoutSeconds", label: "Wallet Sync Timeout Seconds", min: 10 },
      { key: "runTimeoutSeconds", label: "Run Timeout Seconds", min: 10 },
      { key: "baselineLookbackDays", label: "Baseline Lookback Days", min: 1 },
      { key: "scanWindowDays", label: "Scan Window Days", min: 1 },
    ],
  },
  {
    title: "Detection Gates",
    icon: Search,
    fields: [
      { key: "lookalikeRecencyDays", label: "Lookalike Recency Days", min: 1 },
      { key: "lookalikePrefixMin", label: "Lookalike Prefix Min", min: 4 },
      { key: "lookalikeSuffixMin", label: "Lookalike Suffix Min", min: 4 },
      { key: "lookalikeSingleSideMin", label: "Lookalike Single-Side Min", min: 6 },
      { key: "minInjectionCount", label: "Minimum Injection Count", min: 1 },
    ],
  },
];

export default function Settings() {
  const [form, setForm] = useState<SettingsResponse | null>(null);
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiClient.getSettings(),
  });
  const saveSettings = useMutation({
    mutationFn: (payload: SettingsResponse) => apiClient.updateSettings(payload),
    onSuccess: async (next) => {
      setForm(next);
      await queryClient.invalidateQueries({ queryKey: ["settings"] });
    },
  });

  useEffect(() => {
    if (data) {
      setForm(data);
    }
  }, [data]);

  const updateNumber = (key: keyof SettingsResponse, value: string) => {
    setForm((current) => {
      if (!current) {
        return current;
      }
      const parsed = Number(value);
      return { ...current, [key]: Number.isFinite(parsed) ? parsed : 0 };
    });
  };

  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-12">
        <h1 className="text-2xl mb-2 tracking-tight">Settings</h1>
        <p className="text-muted-foreground text-sm">Runtime configuration for new scanner runs</p>
      </div>

      {isLoading ? <div className="mb-8 text-sm text-muted-foreground">Loading backend settings...</div> : null}
      {error ? <div className="mb-8 text-sm text-destructive-foreground">Failed to load settings: {(error as Error).message}</div> : null}
      {saveSettings.error ? <div className="mb-8 text-sm text-destructive-foreground">Failed to save settings: {(saveSettings.error as Error).message}</div> : null}
      {saveSettings.data ? <div className="mb-8 text-sm text-muted-foreground font-mono">Settings saved</div> : null}

      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (form) {
            saveSettings.mutate(form);
          }
        }}
      >
        {settingGroups.map(({ title, icon: Icon, fields }) => (
          <div key={title} className="border border-border mb-12">
            <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
              <Icon className="w-4 h-4 text-muted-foreground" />
              <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">{title}</div>
            </div>
            <div className="p-8 grid md:grid-cols-2 gap-6 text-sm font-mono">
              {fields.map((field) => (
                <label key={field.key} className="grid gap-2">
                  <span className="text-muted-foreground">{field.label}</span>
                  <Input
                    type="number"
                    min={field.min}
                    value={form?.[field.key] ?? ""}
                    onChange={(event) => updateNumber(field.key, event.target.value)}
                    className="font-mono"
                  />
                </label>
              ))}
            </div>
          </div>
        ))}

        <div className="mb-12">
          <Button type="submit" disabled={!form || saveSettings.isPending}>
            <Save className="w-4 h-4" />
            Save
          </Button>
        </div>
      </form>

      <div className="border border-border mt-12">
        <div className="px-8 py-4 border-b border-border bg-muted/30 flex items-center gap-4">
          <Database className="w-4 h-4 text-muted-foreground" />
          <div className="text-sm uppercase tracking-widest font-mono text-muted-foreground">Persistence</div>
        </div>
        <div className="p-8 text-sm text-muted-foreground">
          Saved values are applied to new runs through <span className="font-mono">app_config_overrides</span>.
          <br />
          Environment values remain the fallback when an override is empty.
        </div>
      </div>
    </div>
  );
}
