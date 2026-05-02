import { Outlet, Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Flag,
  ArrowLeftRight,
  Play,
  Wallet,
  Users,
  Download,
  Settings
} from "lucide-react";

export default function AppLayout() {
  const location = useLocation();

  const navItems = [
    { path: "/app", label: "Overview", icon: LayoutDashboard },
    { path: "/app/candidates", label: "Candidates", icon: Flag },
    { path: "/app/transactions", label: "Transactions", icon: ArrowLeftRight },
    { path: "/app/runs", label: "Runs", icon: Play },
    { path: "/app/wallet-sync", label: "Scan Configuration", icon: Wallet },
    { path: "/app/counterparties", label: "Counterparties", icon: Users },
    { path: "/app/exports", label: "Exports", icon: Download },
    { path: "/app/settings", label: "Settings", icon: Settings },
  ];

  const isActive = (path: string) => {
    if (path === "/app") {
      return location.pathname === "/app";
    }
    return location.pathname.startsWith(path);
  };

  return (
    <div className="h-screen flex flex-col md:flex-row bg-background">
      {/* Mobile Header */}
      <div className="md:hidden border-b border-border p-4 flex items-center justify-between">
        <div className="font-mono text-sm">PoisonTrace</div>
        <Link to="/" className="text-xs text-muted-foreground hover:text-foreground transition-colors">
          Exit
        </Link>
      </div>

      {/* Desktop Sidebar */}
      <aside className="hidden md:flex md:flex-col w-64 border-r border-border">
        <div className="p-8 border-b border-border">
          <Link to="/" className="font-mono tracking-tight hover:text-muted-foreground transition-colors">
            PoisonTrace
          </Link>
          <div className="text-xs text-muted-foreground mt-2 font-mono">Monitoring</div>
        </div>
        <nav className="flex-1 p-6 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-3 px-4 py-3 text-sm transition-colors ${
                  active
                    ? "bg-foreground text-background"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/30"
                }`}
              >
                <Icon className="w-4 h-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="p-6 border-t border-border text-xs text-muted-foreground font-mono space-y-1">
          <div>Current scan window</div>
          <div>Last update: 2m ago</div>
        </div>
      </aside>

      {/* Mobile Bottom Nav */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 border-t border-border bg-background grid grid-cols-4 gap-1 p-2">
        <Link
          to="/app"
          className={`flex flex-col items-center gap-1 p-2 text-xs transition-colors ${
            isActive("/app") && location.pathname === "/app"
              ? "text-foreground"
              : "text-muted-foreground"
          }`}
        >
          <LayoutDashboard className="w-5 h-5" />
          Alerts
        </Link>
        <Link
          to="/app/candidates"
          className={`flex flex-col items-center gap-1 p-2 text-xs transition-colors ${
            isActive("/app/candidates") ? "text-foreground" : "text-muted-foreground"
          }`}
        >
          <Flag className="w-5 h-5" />
          Candidates
        </Link>
        <Link
          to="/app/transactions"
          className={`flex flex-col items-center gap-1 p-2 text-xs transition-colors ${
            isActive("/app/transactions") ? "text-foreground" : "text-muted-foreground"
          }`}
        >
          <ArrowLeftRight className="w-5 h-5" />
          Activity
        </Link>
        <Link
          to="/app/settings"
          className={`flex flex-col items-center gap-1 p-2 text-xs transition-colors ${
            isActive("/app/settings") ? "text-foreground" : "text-muted-foreground"
          }`}
        >
          <Settings className="w-5 h-5" />
          Settings
        </Link>
      </nav>

      {/* Main Content */}
      <main className="flex-1 overflow-auto pb-20 md:pb-0">
        <Outlet />
      </main>
    </div>
  );
}
