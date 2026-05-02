import { createBrowserRouter } from "react-router-dom";
import Landing from "./pages/Landing";
import Methodology from "./pages/Methodology";
import AppLayout from "./components/AppLayout";
import Overview from "./pages/app/Overview";
import Candidates from "./pages/app/Candidates";
import Transactions from "./pages/app/Transactions";
import Runs from "./pages/app/Runs";
import WalletSync from "./pages/app/WalletSync";
import Counterparties from "./pages/app/Counterparties";
import Exports from "./pages/app/Exports";
import Settings from "./pages/app/Settings";
import NotFound from "./pages/NotFound";

export const routes = [
  {
    path: "/",
    Component: Landing,
  },
  {
    path: "/methodology",
    Component: Methodology,
  },
  {
    path: "/app",
    Component: AppLayout,
    children: [
      { index: true, Component: Overview },
      { path: "candidates", Component: Candidates },
      { path: "transactions", Component: Transactions },
      { path: "runs", Component: Runs },
      { path: "wallet-sync", Component: WalletSync },
      { path: "counterparties", Component: Counterparties },
      { path: "exports", Component: Exports },
      { path: "settings", Component: Settings },
    ],
  },
  {
    path: "*",
    Component: NotFound,
  },
];

export const router = createBrowserRouter(routes);
