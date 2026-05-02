import { CheckCircle, XCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { getTransactions, type TransactionRow } from "../../lib/api";
import { shortAddress, timeAgo } from "../../lib/format";

export default function Transactions() {
  const [items, setItems] = useState<TransactionRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void getTransactions(1, 100).then((res) => setItems(res.items)).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to load transactions");
    });
  }, []);

  return (
    <div className="flex flex-col h-full">
      <div className="p-8 border-b border-border">
        <h1 className="text-2xl mb-2 tracking-tight">Transactions</h1>
        <p className="text-muted-foreground text-sm">Complete transaction scan with normalization and eligibility status</p>
      </div>
      {error ? <div className="px-8 py-4 text-sm text-destructive-foreground">{error}</div> : null}
      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-muted/30 border-b border-border sticky top-0">
            <tr>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Signature</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Time</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Norm Status</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Eligible</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Relation</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Dust</th>
              <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">Amount Raw</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {items.map((tx) => (
              <tr key={`${tx.signature}-${tx.transferIndex}-${tx.relationType}`} className={`hover:bg-muted/30 transition-colors ${tx.dustStatus === "dust" ? "border-l-2 border-l-destructive-foreground" : ""}`}>
                <td className="px-8 py-5 font-mono text-sm">{shortAddress(tx.signature, 6, 6)}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{timeAgo(tx.blockTime)}</td>
                <td className="px-8 py-5">{tx.normalizationStatus === "resolved" ? <div className="flex items-center gap-2"><CheckCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm">Resolved</span></div> : <div className="flex items-center gap-2"><XCircle className="w-4 h-4 text-muted-foreground" /><span className="text-sm text-destructive-foreground">{tx.normalizationStatus}</span></div>}</td>
                <td className="px-8 py-5 text-sm">{tx.poisoningEligible ? "Yes" : "No"}</td>
                <td className="px-8 py-5 text-sm text-muted-foreground">{tx.relationType}</td>
                <td className="px-8 py-5 text-sm">{tx.dustStatus}</td>
                <td className="px-8 py-5 font-mono text-sm text-muted-foreground">{tx.amountRaw}</td>
              </tr>
            ))}
            {items.length === 0 ? <tr><td className="px-8 py-5 text-sm text-muted-foreground" colSpan={7}>No transactions found.</td></tr> : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
