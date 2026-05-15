import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

describe("counterparties route", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (url.includes("/api/counterparties")) {
        return new Response(JSON.stringify({
          items: [
            {
              focalWallet: "wallet11111111111111111111111111111111",
              counterpartyAddress: "counterparty1111111111111111111111111111",
              firstSeenAt: "2026-05-01T10:00:00Z",
              lastSeenAt: "2026-05-02T10:00:00Z",
              inboundCount: 3,
              outboundCount: 2,
              lastOutboundAt: "2026-05-02T09:00:00Z",
              candidateLinks: 1,
            },
          ],
          total: 1,
          page: 1,
          pageSize: 25,
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
    }) as unknown as typeof fetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders counterparties endpoint fields", async () => {
    const router = createMemoryRouter(routes, { initialEntries: ["/app/counterparties"] });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);

    expect(await screen.findByRole("heading", { name: "Counterparties" })).toBeInTheDocument();
    expect(await screen.findByText("3")).toBeInTheDocument();
    expect(await screen.findByText("2")).toBeInTheDocument();
    expect(await screen.findByText("1")).toBeInTheDocument();
  });
});
