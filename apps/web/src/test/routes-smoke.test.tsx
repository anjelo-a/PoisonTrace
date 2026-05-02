import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { routes } from "../app/routes";

vi.stubGlobal("fetch", vi.fn(async (url: string) => {
  if (url.includes("/api/overview")) {
    return new Response(JSON.stringify({ metrics: { candidatesEmitted: 0, unknownGateBlocks: 0, transactionsScanned: 0, passedTransactions: 0, lastScanUpdateAt: "", scanWindowLabel: "24h", passRatePct: 0 }, recentCandidates: [] }), { status: 200 });
  }
  if (url.includes("/api/candidates")) {
    return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 25 }), { status: 200 });
  }
  return new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }), { status: 200 });
}) as unknown as typeof fetch);

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);
}

describe("route smoke", () => {
  it("renders overview route", async () => {
    renderAt("/app");
    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
  });

  it("renders candidates route", async () => {
    renderAt("/app/candidates");
    expect(await screen.findByRole("heading", { name: "Candidates" })).toBeInTheDocument();
  });
});
