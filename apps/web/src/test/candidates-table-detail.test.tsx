import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

describe("candidates table/detail", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (url: string) => {
      if (url.includes("/api/candidates")) {
        return new Response(JSON.stringify({
          items: [
            {
              walletSyncRunId: 1,
              focalWallet: "wallet",
              signature: "abcdef1234567890",
              transferIndex: 0,
              blockTime: "2026-05-02T10:00:00Z",
              suspiciousCounterparty: "sus123456",
              matchedLegitCounterparty: "leg123456",
              repeatInjectionCount: 2,
              recencyDays: 1,
            }
          ],
          total: 1,
          page: 1,
          pageSize: 25
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ metrics: { candidatesEmitted: 1, unknownGateBlocks: 0, transactionsScanned: 10, passedTransactions: 8, lastScanUpdateAt: "2026-05-02T10:00:00Z", scanWindowLabel: "7d", passRatePct: 80 }, recentCandidates: [] }), { status: 200 });
    }) as unknown as typeof fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders table row and opens detail drawer", async () => {
    const router = createMemoryRouter(routes, { initialEntries: ["/app/candidates"] });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);

    const rowSig = await screen.findByText(/abcdef/i);
    await userEvent.click(rowSig);
    expect(await screen.findByText("Candidate Detail")).toBeInTheDocument();
    expect(screen.getByText(/Unknown-gate blocked events are excluded/i)).toBeInTheDocument();
  });
});
