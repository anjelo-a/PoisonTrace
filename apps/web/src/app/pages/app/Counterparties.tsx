import { AlertTriangle } from "lucide-react";

export default function Counterparties() {
  const counterparties = [
    { address: "9xyz...4abc", firstSeen: "2h ago", lastSeen: "2m ago", inboundCount: 3, outboundCount: 0, lastOutboundAt: null, candidateLinks: 1 },
    { address: "2def...8xyz", firstSeen: "5h ago", lastSeen: "5m ago", inboundCount: 5, outboundCount: 0, lastOutboundAt: null, candidateLinks: 1 },
    { address: "3Vwx...9Yza", firstSeen: "14d ago", lastSeen: "1h ago", inboundCount: 8, outboundCount: 4, lastOutboundAt: "3d ago", candidateLinks: 0 },
    { address: "7mno...1pqr", firstSeen: "12h ago", lastSeen: "12m ago", inboundCount: 2, outboundCount: 0, lastOutboundAt: null, candidateLinks: 1 },
    { address: "7Hij...3Klm", firstSeen: "9d ago", lastSeen: "4h ago", inboundCount: 6, outboundCount: 2, lastOutboundAt: "7d ago", candidateLinks: 0 },
    { address: "5stu...6vwx", firstSeen: "18h ago", lastSeen: "18m ago", inboundCount: 4, outboundCount: 0, lastOutboundAt: null, candidateLinks: 1 },
    { address: "1Tuv...7Wxy", firstSeen: "21d ago", lastSeen: "6h ago", inboundCount: 18, outboundCount: 6, lastOutboundAt: "2d ago", candidateLinks: 0 },
    { address: "4yza...3bcd", firstSeen: "25h ago", lastSeen: "25m ago", inboundCount: 6, outboundCount: 0, lastOutboundAt: null, candidateLinks: 1 },
  ];

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Counterparties</h1>
        <p className="text-muted-foreground text-sm">Address interaction history with deterministic metadata</p>
      </div>

      {/* Summary */}
      <div className="px-8 py-6 border-b border-border grid grid-cols-4 gap-12">
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Total Counterparties</div>
          <div className="text-2xl font-mono tracking-tight">147</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Inbound Only</div>
          <div className="text-2xl font-mono tracking-tight">68</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">With Candidate Links</div>
          <div className="text-2xl font-mono tracking-tight">23</div>
        </div>
        <div>
          <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">New (24h)</div>
          <div className="text-2xl font-mono tracking-tight">5</div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-8 py-4 border-b border-border flex gap-6">
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>All Interaction Types</option>
          <option>Inbound Only</option>
          <option>Outbound Present</option>
        </select>
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>All Time</option>
          <option>Last 24h</option>
          <option>Last 7d</option>
          <option>Last 30d</option>
        </select>
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>Sort: Most Recent</option>
          <option>Sort: First Seen</option>
          <option>Sort: Inbound Count</option>
        </select>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Address
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                First Seen
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Last Seen
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Inbound Count
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Outbound Count
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Last Outbound At
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Candidate Links
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {counterparties.map((cp) => (
              <tr
                key={cp.address}
                className={`hover:bg-muted/30 transition-colors ${cp.candidateLinks > 0 ? "border-l-2 border-l-destructive-foreground" : ""}`}
              >
                <td className="px-8 py-5 font-mono text-sm">{cp.address}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{cp.firstSeen}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{cp.lastSeen}</td>
                <td className="px-8 py-5 text-sm font-mono">{cp.inboundCount}</td>
                <td className="px-8 py-5 text-sm font-mono text-muted-foreground">{cp.outboundCount}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">
                  {cp.lastOutboundAt || "—"}
                </td>
                <td className="px-8 py-5">
                  {cp.candidateLinks > 0 ? (
                    <div className="flex items-center gap-2">
                      <AlertTriangle className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm">{cp.candidateLinks}</span>
                    </div>
                  ) : (
                    <span className="text-sm text-muted-foreground">0</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground">
        <div className="font-mono">Showing 8 of 147 counterparties</div>
        <div className="flex gap-3">
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">Previous</button>
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">Next</button>
        </div>
      </div>
    </div>
  );
}
