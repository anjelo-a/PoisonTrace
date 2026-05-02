import { useState } from "react";
import { X, AlertTriangle, ExternalLink } from "lucide-react";

type CandidateItem = {
  id: string;
  signature: string;
  blockTime: string;
  timeAgo: string;
  suspiciousCounterparty: string;
  matchedLegitCounterparty: string | null;
  repeatInjectionCount: number;
  recencyDays: number;
  unknownGateReason: string | null;
};

const mockCandidates: CandidateItem[] = [
  {
    id: "1",
    signature: "5KqRn8vG...Hs4d8Hs",
    blockTime: "2026-05-01 14:32:18",
    timeAgo: "2m ago",
    suspiciousCounterparty: "9xyz...4abc",
    matchedLegitCounterparty: "9xya...7def",
    repeatInjectionCount: 3,
    recencyDays: 0,
    unknownGateReason: null,
  },
  {
    id: "2",
    signature: "9Lmn7kPq...Ty3x3Ty",
    blockTime: "2026-05-01 14:29:42",
    timeAgo: "5m ago",
    suspiciousCounterparty: "2def...8xyz",
    matchedLegitCounterparty: "2deg...1abc",
    repeatInjectionCount: 5,
    recencyDays: 0,
    unknownGateReason: null,
  },
  {
    id: "3",
    signature: "1AbcDef2...Qw7p7Qw",
    blockTime: "2026-05-01 14:22:55",
    timeAgo: "12m ago",
    suspiciousCounterparty: "7mno...1pqr",
    matchedLegitCounterparty: null,
    repeatInjectionCount: 2,
    recencyDays: 0,
    unknownGateReason: null,
  },
  {
    id: "4",
    signature: "7ZxyKlm9...Mn4k4Mn",
    blockTime: "2026-05-01 14:16:33",
    timeAgo: "18m ago",
    suspiciousCounterparty: "5stu...6vwx",
    matchedLegitCounterparty: "5stu...2abc",
    repeatInjectionCount: 4,
    recencyDays: 1,
    unknownGateReason: null,
  },
  {
    id: "5",
    signature: "3DefGhi5...Ij9r9Ij",
    blockTime: "2026-05-01 14:09:21",
    timeAgo: "25m ago",
    suspiciousCounterparty: "4yza...3bcd",
    matchedLegitCounterparty: "4yza...9efg",
    repeatInjectionCount: 6,
    recencyDays: 0,
    unknownGateReason: null,
  },
];

export default function Candidates() {
  const [selectedCandidate, setSelectedCandidate] = useState<CandidateItem | null>(null);

  return (
    <div className="h-full flex">
      {/* List View */}
      <div className={`flex-1 flex flex-col ${selectedCandidate ? "hidden md:flex" : ""}`}>
        <div className="p-8 border-b border-border">
          <h1 className="text-2xl mb-2 tracking-tight">Candidates</h1>
          <p className="text-muted-foreground text-sm">Emitted candidates requiring review</p>
        </div>

        {/* Filters */}
        <div className="px-8 py-4 border-b border-border flex gap-6">
          <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
            <option>All Time</option>
            <option>Last 24h</option>
            <option>Last 7d</option>
            <option>Last 30d</option>
          </select>
          <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
            <option>All Recency</option>
            <option>&lt; 1 day</option>
            <option>&lt; 7 days</option>
          </select>
          <select className="px-4 py-2 bg-transparent border border-border text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors focus:outline-none focus:border-foreground">
            <option>Sort: Latest First</option>
            <option>Sort: Oldest First</option>
            <option>Sort: Repeat Count</option>
          </select>
        </div>

        {/* Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full">
            <thead className="bg-muted/30 border-b border-border sticky top-0">
              <tr>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Signature
                </th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Block Time
                </th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Suspicious Counterparty
                </th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Matched Legit
                </th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Repeat Count
                </th>
                <th className="px-8 py-4 text-left text-xs font-mono uppercase tracking-widest text-muted-foreground">
                  Recency Days
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {mockCandidates.map((candidate) => (
                <tr
                  key={candidate.id}
                  onClick={() => setSelectedCandidate(candidate)}
                  className={`cursor-pointer hover:bg-muted/30 transition-colors ${
                    selectedCandidate?.id === candidate.id ? "bg-muted/30" : ""
                  } ${candidate.matchedLegitCounterparty ? "border-l-2 border-l-destructive-foreground" : ""}`}
                >
                  <td className="px-8 py-5">
                    <div className="font-mono text-sm">{candidate.signature}</div>
                    <div className="text-xs text-muted-foreground mt-1">{candidate.timeAgo}</div>
                  </td>
                  <td className="px-8 py-5 text-sm text-muted-foreground font-mono">{candidate.blockTime}</td>
                  <td className="px-8 py-5">
                    <div className="font-mono text-sm">{candidate.suspiciousCounterparty}</div>
                  </td>
                  <td className="px-8 py-5">
                    {candidate.matchedLegitCounterparty ? (
                      <div className="font-mono text-sm text-destructive-foreground">{candidate.matchedLegitCounterparty}</div>
                    ) : (
                      <div className="text-sm text-muted-foreground">—</div>
                    )}
                  </td>
                  <td className="px-8 py-5 text-sm font-mono">{candidate.repeatInjectionCount}</td>
                  <td className="px-8 py-5 text-sm text-muted-foreground">{candidate.recencyDays}d</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="px-8 py-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground">
          <div className="font-mono">Showing 1-5 of 7 candidates</div>
          <div className="flex gap-3">
            <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs" disabled>
              Previous
            </button>
            <button className="px-4 py-2 border border-border hover:border-foreground hover:text-foreground transition-colors text-xs">Next</button>
          </div>
        </div>
      </div>

      {/* Detail Panel */}
      {selectedCandidate && (
        <div className="w-full md:w-[600px] border-l border-border flex flex-col">
          {/* Header */}
          <div className="p-8 border-b border-border flex items-start justify-between">
            <div className="flex-1">
              <div className="text-xs uppercase tracking-widest text-muted-foreground font-mono mb-2">Candidate Detail</div>
              <div className="font-mono text-sm break-all">
                {selectedCandidate.signature}
              </div>
            </div>
            <button
              onClick={() => setSelectedCandidate(null)}
              className="ml-6 p-2 hover:bg-muted/30 transition-colors"
            >
              <X className="w-4 h-4 text-muted-foreground" />
            </button>
          </div>

          {/* Scrollable Content */}
          <div className="flex-1 overflow-auto p-8 space-y-12">
            {/* Summary */}
            <div>
              <h3 className="mb-6 text-sm uppercase tracking-widest text-muted-foreground font-mono">Summary</h3>
              <div className="space-y-3 text-sm font-mono">
                <div className="grid grid-cols-[160px_1fr]">
                  <span className="text-muted-foreground">Block Time:</span>
                  <span>{selectedCandidate.blockTime}</span>
                </div>
                <div className="grid grid-cols-[160px_1fr]">
                  <span className="text-muted-foreground">Suspicious Counterparty:</span>
                  <span>{selectedCandidate.suspiciousCounterparty}</span>
                </div>
                {selectedCandidate.matchedLegitCounterparty && (
                  <div className="grid grid-cols-[160px_1fr]">
                    <span className="text-muted-foreground">Matched Legit:</span>
                    <span className="text-destructive-foreground">{selectedCandidate.matchedLegitCounterparty}</span>
                  </div>
                )}
                <div className="grid grid-cols-[160px_1fr]">
                  <span className="text-muted-foreground">Repeat Injections:</span>
                  <span>{selectedCandidate.repeatInjectionCount} events</span>
                </div>
                <div className="grid grid-cols-[160px_1fr]">
                  <span className="text-muted-foreground">Recency:</span>
                  <span>{selectedCandidate.recencyDays} days</span>
                </div>
                {selectedCandidate.unknownGateReason && (
                  <div className="grid grid-cols-[160px_1fr]">
                    <span className="text-muted-foreground">Unknown Gate:</span>
                    <span className="text-destructive-foreground">{selectedCandidate.unknownGateReason}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Gate Explanations */}
            <div>
              <h3 className="mb-6 text-sm uppercase tracking-widest text-muted-foreground font-mono">Gate Evaluation</h3>
              <p className="text-xs text-muted-foreground mb-6 font-mono">
                Illustrative example, not production thresholds.
              </p>
              <div className="space-y-6">
                {/* Gate 1 - Lookalike */}
                <div className="border border-border">
                  <div className="px-6 py-4 bg-muted/30 border-b border-border flex items-center justify-between">
                    <span className="text-sm">Gate: Lookalike Similarity</span>
                    <span className="text-xs font-mono text-destructive-foreground">FAIL</span>
                  </div>
                  <div className="p-6 space-y-3 text-sm font-mono">
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Observed:</span>
                      <span className="text-foreground">3 character difference</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Expected:</span>
                      <span className="text-foreground">&gt; 10 char diff</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Outcome:</span>
                      <span className="text-destructive-foreground">Probable lookalike</span>
                    </div>
                  </div>
                </div>

                {/* Gate 2 - Min Injections */}
                <div className="border border-border">
                  <div className="px-6 py-4 bg-muted/30 border-b border-border flex items-center justify-between">
                    <span className="text-sm">Gate: Min Repeat Injections</span>
                    <span className="text-xs font-mono text-destructive-foreground">FAIL</span>
                  </div>
                  <div className="p-6 space-y-3 text-sm font-mono">
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Observed:</span>
                      <span className="text-foreground">{selectedCandidate.repeatInjectionCount} qualifying events</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Expected:</span>
                      <span className="text-foreground">&lt; 2 events (threshold)</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr] text-muted-foreground">
                      <span>Outcome:</span>
                      <span className="text-destructive-foreground">Repeat pattern detected</span>
                    </div>
                  </div>
                </div>

                {/* Gate 3 - Recency */}
                <div className="border border-border opacity-60">
                  <div className="px-6 py-4 bg-muted/30 border-b border-border flex items-center justify-between">
                    <span className="text-sm">Gate: Recency Window</span>
                    <span className="text-xs font-mono text-muted-foreground">PASS</span>
                  </div>
                  <div className="p-6 space-y-3 text-sm font-mono text-muted-foreground">
                    <div className="grid grid-cols-[100px_1fr]">
                      <span>Observed:</span>
                      <span>{selectedCandidate.recencyDays}d ago</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr]">
                      <span>Expected:</span>
                      <span>&lt; 7d (active window)</span>
                    </div>
                    <div className="grid grid-cols-[100px_1fr]">
                      <span>Outcome:</span>
                      <span>Within active window</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Conclusion */}
            <div className="border border-border bg-muted/30 p-6">
              <h3 className="mb-3 text-sm">Detection Result</h3>
              <p className="text-sm text-muted-foreground font-mono leading-relaxed">
                2 gates failed, all required gates resolved → Candidate emitted
              </p>
            </div>

            {/* Related Evidence */}
            <div>
              <h3 className="mb-6 text-sm uppercase tracking-widest text-muted-foreground font-mono">Related Evidence</h3>
              <div className="space-y-3">
                <button className="w-full p-4 border border-border hover:border-foreground transition-colors flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Transaction detail</span>
                  <ExternalLink className="w-4 h-4 text-muted-foreground" />
                </button>
                <button className="w-full p-4 border border-border hover:border-foreground transition-colors flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Counterparty profile</span>
                  <ExternalLink className="w-4 h-4 text-muted-foreground" />
                </button>
                <button className="w-full p-4 border border-border hover:border-foreground transition-colors flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Detection run log</span>
                  <ExternalLink className="w-4 h-4 text-muted-foreground" />
                </button>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="p-6 border-t border-border flex gap-3">
            <button className="flex-1 px-6 py-3 border border-border hover:border-foreground hover:text-foreground transition-colors text-sm">
              Mark Reviewed
            </button>
            <button className="flex-1 px-6 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors text-sm">
              Export Report
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
