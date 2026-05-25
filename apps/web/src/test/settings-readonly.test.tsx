import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

describe("settings editor", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (url.includes("/api/settings")) {
        return new Response(JSON.stringify({
          maxWalletsPerRun: 12,
          maxTXPagesPerWallet: 34,
          maxTXPerWallet: 56,
          maxConcurrentWallets: 4,
          walletSyncTimeoutSeconds: 120,
          runTimeoutSeconds: 900,
          maxHeliusRetries: 5,
          heliusRequestDelayMS: 300,
          baselineLookbackDays: 30,
          scanWindowDays: 7,
          lookalikeRecencyDays: 14,
          lookalikePrefixMin: 4,
          lookalikeSuffixMin: 4,
          lookalikeSingleSideMin: 6,
          minInjectionCount: 2,
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
    }) as unknown as typeof fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("shows backend values and save action", async () => {
    const router = createMemoryRouter(routes, { initialEntries: ["/app/settings"] });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);

    expect(await screen.findByText("Runtime configuration for new scanner runs")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("12")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("34")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("900")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
    expect(await screen.findByText(/app_config_overrides/i)).toBeInTheDocument();
  });
});
