import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

describe("exports deep-link", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (url.includes("/api/reports/candidates")) {
        return new Response(JSON.stringify({
          items: [
            {
              walletSyncRunId: 9,
              runId: 77,
              focalWallet: "wallet",
              signature: "sig1234567890",
              transferIndex: 2,
              blockTime: "2026-05-02T10:00:00Z",
              suspiciousCounterparty: "sus123456",
              matchedLegitCounterparty: "leg123456",
            },
          ],
          total: 1,
          page: 1,
          pageSize: 25,
        }), { status: 200 });
      }
      if (url.includes("/api/exports/files")) {
        return new Response(JSON.stringify({ runId: 77, files: [] }), { status: 200 });
      }
      if (url.includes("/api/exports/generate")) {
        return new Response(JSON.stringify({ runId: 77, outDir: "artifacts/web_exports/run_77", schemaVersion: "phase5-v1", generatedAt: "2026-05-02T10:00:00Z", files: [] }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
    }) as unknown as typeof fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("builds candidate detail link with stable identity params", async () => {
    const router = createMemoryRouter(routes, { initialEntries: ["/app/exports?run_id=77"] });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);

    const link = await screen.findByRole("link", { name: "Open Detail" });
    expect(link.getAttribute("href")).toContain("/app/candidates?wallet_sync_run_id=9&signature=sig1234567890&transfer_index=2");
  });
});
