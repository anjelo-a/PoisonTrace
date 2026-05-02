import { CheckCircle, XCircle } from "lucide-react";

export default function Transactions() {
  const transactions = [
    { sig: "8Pqr...5Stu", time: "1m ago", normStatus: "Resolved", eligible: "Yes", relationType: "receiver", dustStatus: "No", assetType: "SOL", amountRaw: "2500000000" },
    { sig: "5KqR...8Hs", time: "2m ago", normStatus: "Resolved", eligible: "Yes", relationType: "receiver", dustStatus: "Yes", assetType: "SOL", amountRaw: "1000" },
    { sig: "2Bcd...1Efg", time: "4m ago", normStatus: "Resolved", eligible: "No", relationType: "sender", dustStatus: "No", assetType: "SOL", amountRaw: "1200000000" },
    { sig: "9Lmn...3Ty", time: "5m ago", normStatus: "Resolved", eligible: "Yes", relationType: "receiver", dustStatus: "Yes", assetType: "SOL", amountRaw: "5000" },
    { sig: "4Nop...6Qrs", time: "8m ago", normStatus: "Resolved", eligible: "Yes", relationType: "receiver", dustStatus: "No", assetType: "SOL", amountRaw: "5000000000" },
    { sig: "1Abc...7Qw", time: "12m ago", normStatus: "Resolved", eligible: "Yes", relationType: "receiver", dustStatus: "Yes", assetType: "SOL", amountRaw: "20000" },
    { sig: "6Zab...2Cde", time: "15m ago", normStatus: "Failed", eligible: "No", relationType: "sender", dustStatus: "No", assetType: "Unknown", amountRaw: "—" },
  ];

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Transactions</h1>
        <p className="text-muted-foreground text-sm">Complete transaction scan with normalization and eligibility status</p>
      </div>

      {/* Filters */}
      <div className="px-8 py-4 border-b border-border flex gap-6">
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>All Normalization Status</option>
          <option>Resolved</option>
          <option>Failed</option>
        </select>
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>All Eligible</option>
          <option>Eligible Only</option>
          <option>Not Eligible</option>
        </select>
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>All Relation Types</option>
          <option>Receiver</option>
          <option>Sender</option>
        </select>
        <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
          <option>Last 24h</option>
          <option>Last 7d</option>
          <option>Last 30d</option>
          <option>All Time</option>
        </select>
      </div>

      {/* Transaction List */}
      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Signature
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Time
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Norm Status
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Eligible
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Relation Type
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Dust Status
              </th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                Amount Raw
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {transactions.map((tx) => (
              <tr
                key={tx.sig}
                className={`hover:bg-muted/30 transition-colors ${tx.dustStatus === "Yes" ? "border-l-2 border-l-destructive-foreground" : ""}`}
              >
                <td className="px-8 py-5 font-mono text-sm">{tx.sig}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{tx.time}</td>
                <td className="px-8 py-5">
                  {tx.normStatus === "Resolved" ? (
                    <div className="flex items-center gap-2">
                      <CheckCircle className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm">Resolved</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <XCircle className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm text-destructive-foreground">Failed</span>
                    </div>
                  )}
                </td>
                <td className="px-8 py-5 text-sm">{tx.eligible}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{tx.relationType}</td>
                <td className="px-8 py-5 text-sm">{tx.dustStatus}</td>
                <td className="px-8 py-5 font-mono text-sm text-muted-foreground">{tx.amountRaw}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Summary Bar */}
      <div className="px-8 py-4 border-t border-border flex items-center justify-between">
        <div className="text-sm text-muted-foreground font-mono">
          Showing 7 transactions • 5 eligible • 2 dust detected
        </div>
        <div className="flex gap-3">
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">
            Previous
          </button>
          <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">
            Next
          </button>
        </div>
      </div>
    </div>
  );
}
