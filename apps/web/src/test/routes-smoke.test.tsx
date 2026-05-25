import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);
}

describe("route smoke", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (url.includes("/api/overview")) {
        return new Response(JSON.stringify({ metrics: { candidatesEmitted: 0, unknownGateBlocks: 0, transactionsScanned: 0, passedTransactions: 0, lastScanUpdateAt: "", scanWindowLabel: "7d", passRatePct: 0 }, recentCandidates: [] }), { status: 200 });
      }
      if (url.includes("/api/runs")) {
        return new Response(JSON.stringify({ items: [{ runId: 42, status: "succeeded", startedAt: "2026-05-01T00:00:00Z", completedAt: "2026-05-01T01:00:00Z", walletsProcessed: 2, walletsFailed: 0, walletsSkipped: 0, incompleteWindows: 0, truncationWalletRate: "0", unknownGatePresent: false }], total: 1, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/wallet-sync")) {
        return new Response(JSON.stringify({ items: [{ walletSyncRunId: 1, runId: 42, focalWallet: "wallet", status: "succeeded", baselineStartAt: "2026-04-01T00:00:00Z", baselineEndAt: "2026-04-30T00:00:00Z", scanStartAt: "2026-05-01T00:00:00Z", scanEndAt: "2026-05-02T00:00:00Z", baselineComplete: true, incompleteWindow: false, unknownGateReason: "", transactionsFetched: 10, truncationReason: "", updatedAt: "2026-05-02T00:00:00Z" }], total: 1, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/candidates")) {
        return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/counterparties")) {
        return new Response(JSON.stringify({ items: [{ focalWallet: "wallet", counterpartyAddress: "counterparty", firstSeenAt: "2026-05-01T00:00:00Z", lastSeenAt: "2026-05-02T00:00:00Z", inboundCount: 1, outboundCount: 2, lastOutboundAt: "2026-05-02T00:00:00Z", candidateLinks: 0 }], total: 1, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/reports/wallets")) {
        return new Response(JSON.stringify({ items: [{ runId: 42, walletSyncRunId: 1, focalWallet: "wallet", candidateCount: 0, unknownGateBlockCount: 0, incompleteWindow: false, unknownGateReason: "", truncationReason: "", baselineComplete: true, scanStartAt: "2026-05-01T00:00:00Z", scanEndAt: "2026-05-02T00:00:00Z", baselineStartAt: "2026-04-01T00:00:00Z", baselineEndAt: "2026-04-30T00:00:00Z", transactionsFetched: 10, sourceReferences: { walletSyncRunId: 1, runId: 42 } }], total: 1, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/reports/candidates")) {
        return new Response(JSON.stringify({ items: [{ walletSyncRunId: 1, runId: 42, focalWallet: "wallet", signature: "sig", transferIndex: 0, blockTime: "2026-05-02T00:00:00Z", suspiciousCounterparty: "sus", matchedLegitCounterparty: "legit" }], total: 26, page: 1, pageSize: 25 }), { status: 200 });
      }
      if (url.includes("/api/exports/files")) {
        return new Response(JSON.stringify({ runId: 42, files: [{ name: "report_manifest.json", sizeBytes: 1234, modifiedAt: "2026-05-02T00:00:00Z", downloadUrl: "/api/exports/download?run_id=42&file=report_manifest.json" }] }), { status: 200 });
      }
      if (url.includes("/api/exports/generate")) {
        return new Response(JSON.stringify({ runId: 42, outDir: "artifacts/web_exports/run_42", schemaVersion: "phase5-v1", generatedAt: "2026-05-02T00:00:00Z", files: [] }), { status: 200 });
      }
      if (url.includes("/api/settings")) {
        return new Response(JSON.stringify({ maxWalletsPerRun: 10, maxTXPagesPerWallet: 10, maxTXPerWallet: 1000, maxConcurrentWallets: 2, walletSyncTimeoutSeconds: 120, runTimeoutSeconds: 1200, maxHeliusRetries: 5, heliusRequestDelayMS: 250, baselineLookbackDays: 30, scanWindowDays: 7, lookalikeRecencyDays: 14, lookalikePrefixMin: 4, lookalikeSuffixMin: 4, lookalikeSingleSideMin: 6, minInjectionCount: 2 }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
    }) as unknown as typeof fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders overview route", async () => {
    renderAt("/app");
    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
  });

  it("renders candidates route", async () => {
    renderAt("/app/candidates");
    expect(await screen.findByRole("heading", { name: "Candidates" })).toBeInTheDocument();
  });

  it("renders runs route", async () => {
    renderAt("/app/runs");
    expect(await screen.findByRole("heading", { name: "Detection Runs" })).toBeInTheDocument();
    expect(await screen.findByText("run-42")).toBeInTheDocument();
  });

  it("renders wallet sync route", async () => {
    renderAt("/app/wallet-sync");
    expect(await screen.findByRole("heading", { name: "Scan Configuration" })).toBeInTheDocument();
    expect(await screen.findByText("wallet")).toBeInTheDocument();
  });

  it("renders counterparties route", async () => {
    renderAt("/app/counterparties");
    expect(await screen.findByRole("heading", { name: "Counterparties" })).toBeInTheDocument();
    expect(await screen.findByText("Counterparty")).toBeInTheDocument();
  });

  it("renders wallet reports route", async () => {
    renderAt("/app/reports/wallets?run_id=42");
    expect(await screen.findByRole("heading", { name: "Wallet Inspection Reports" })).toBeInTheDocument();
    expect(await screen.findByText("wallet")).toBeInTheDocument();
  });

  it("renders exports route and paginates", async () => {
    renderAt("/app/exports?run_id=42");
    expect(await screen.findByRole("heading", { name: "Reports and Exports" })).toBeInTheDocument();
    expect(await screen.findByText("Open Detail")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Previous" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("renders settings route", async () => {
    renderAt("/app/settings");
    expect(await screen.findByRole("heading", { name: "Settings" })).toBeInTheDocument();
    expect(await screen.findByText(/runtime configuration for new scanner runs/i)).toBeInTheDocument();
  });
});
