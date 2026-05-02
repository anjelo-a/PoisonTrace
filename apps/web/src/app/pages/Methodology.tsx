import { Link } from "react-router-dom";

export default function Methodology() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-border bg-background">
        <div className="max-w-7xl mx-auto px-8 py-6 flex items-center justify-between">
          <Link to="/" className="font-mono tracking-tight hover:text-muted-foreground transition-colors">
            PoisonTrace
          </Link>
          <nav className="flex items-center gap-12">
            <Link to="/methodology" className="text-foreground text-sm underline underline-offset-4">
              Methodology
            </Link>
            <Link to="/app/candidates" className="text-muted-foreground hover:text-foreground transition-colors text-sm">
              Review Patterns
            </Link>
          </nav>
        </div>
      </header>

      {/* Content */}
      <div className="max-w-4xl mx-auto px-8 py-16">
        <div className="mb-12">
          <Link to="/" className="text-sm text-muted-foreground hover:text-foreground transition-colors">
            ← Back
          </Link>
        </div>

        <h1 className="text-4xl mb-8 tracking-tight">Detection Methodology</h1>
        <p className="text-muted-foreground leading-relaxed mb-8">
          How PoisonTrace detects probable poisoning patterns in transaction data, what the rules measure,
          and what happens when data is missing.
        </p>

        {/* Scope Statement */}
        <div className="border border-border bg-muted/30 p-6 mb-16">
          <h3 className="mb-3 text-sm uppercase tracking-widest font-mono text-muted-foreground">Scope Statement</h3>
          <p className="text-sm leading-relaxed">
            PoisonTrace identifies probable poisoning patterns in Solana transaction data using rule-based detection.
            It does not confirm scams, perform victim attribution, or guarantee protection from all attack types.
            Flagged patterns are surfaced for review—you decide next action.
          </p>
        </div>

        <div className="space-y-20">
          {/* Overview */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">Overview</h2>
            <div className="space-y-4 text-muted-foreground leading-relaxed">
              <p>
                PoisonTrace uses rule-based detection to identify probable Solana poisoning patterns in transaction datasets.
                Each transaction passes through a series of checks called "gates." When multiple gates fail,
                the transaction is flagged as a probable poisoning signal for your review.
              </p>
              <p>
                This page explains what each gate measures, the thresholds used, and how the system handles
                missing or uncertain data.
              </p>
            </div>
          </section>

          {/* Detection Gates */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">Detection Gates</h2>
            <p className="text-sm text-muted-foreground mb-8 font-mono">
              Illustrative examples, not production thresholds. Production values are tuned based on observed attack patterns and may change.
            </p>
            <div className="space-y-6">
              <div className="border border-border">
                <div className="border-b border-border p-6 bg-muted/30">
                  <h3>Gate 1: Lookalike Counterparty Detection</h3>
                </div>
                <div className="p-6 space-y-3 text-sm font-mono">
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>What it checks:</span>
                    <span className="text-foreground">Visual similarity to previously observed addresses in dataset</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Measurement:</span>
                    <span className="text-foreground">Character difference (Levenshtein distance)</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Threshold:</span>
                    <span className="text-foreground">&gt; 10 character difference expected</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Fails when:</span>
                    <span className="text-destructive-foreground">Address appears visually similar to known counterparty</span>
                  </div>
                </div>
                <div className="px-6 pb-6 text-xs text-muted-foreground leading-relaxed">
                  <span className="text-foreground">Why this matters:</span> Scammers create addresses that look almost
                  identical to legitimate ones, hoping victims will copy the wrong address from transaction history.
                </div>
              </div>

              <div className="border border-border">
                <div className="border-b border-border p-6 bg-muted/30">
                  <h3>Gate 2: Dust Amount Detection</h3>
                </div>
                <div className="p-6 space-y-3 text-sm font-mono">
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>What it checks:</span>
                    <span className="text-foreground">Transaction amount (incoming transfers)</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Measurement:</span>
                    <span className="text-foreground">SOL amount received</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Threshold:</span>
                    <span className="text-foreground">&lt; 0.001 SOL triggers dust flag</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Fails when:</span>
                    <span className="text-destructive-foreground">Tiny unsolicited amounts (dust) detected</span>
                  </div>
                </div>
                <div className="px-6 pb-6 text-xs text-muted-foreground leading-relaxed">
                  <span className="text-foreground">Why this matters:</span> Poisoning attacks often send tiny amounts
                  to pollute transaction history with malicious addresses, making them appear in recent activity.
                </div>
              </div>

              <div className="border border-border">
                <div className="border-b border-border p-6 bg-muted/30">
                  <h3>Gate 3: First-Time Counterparty Check</h3>
                </div>
                <div className="p-6 space-y-3 text-sm font-mono">
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>What it checks:</span>
                    <span className="text-foreground">Prior interaction history with address in dataset</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Measurement:</span>
                    <span className="text-foreground">Number of previous transactions observed</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Threshold:</span>
                    <span className="text-foreground">&gt; 0 prior interactions expected for safe pattern</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Fails when:</span>
                    <span className="text-destructive-foreground">First-time address + other signals present</span>
                  </div>
                </div>
                <div className="px-6 pb-6 text-xs text-muted-foreground leading-relaxed">
                  <span className="text-foreground">Why this matters:</span> Legitimate counterparties usually have observable
                  transaction history. First contact combined with dust or lookalike patterns increases probability of poisoning.
                </div>
              </div>

              <div className="border border-border">
                <div className="border-b border-border p-6 bg-muted/30">
                  <h3>Gate 4: Rapid-Fire Timing Pattern</h3>
                </div>
                <div className="p-6 space-y-3 text-sm font-mono">
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>What it checks:</span>
                    <span className="text-foreground">Time gap between consecutive transactions</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Measurement:</span>
                    <span className="text-foreground">Seconds elapsed since previous tx in sequence</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Threshold:</span>
                    <span className="text-foreground">&gt; 5s expected (human pace)</span>
                  </div>
                  <div className="grid grid-cols-[140px_1fr] text-muted-foreground">
                    <span>Fails when:</span>
                    <span className="text-destructive-foreground">Automated burst detected (&lt; 5s gaps)</span>
                  </div>
                </div>
                <div className="px-6 pb-6 text-xs text-muted-foreground leading-relaxed">
                  <span className="text-foreground">Why this matters:</span> Scam bots send multiple transactions
                  in rapid succession. Normal activity has more spacing between transactions.
                </div>
              </div>
            </div>
          </section>

          {/* Flagging Logic */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">When Patterns Get Flagged</h2>
            <div className="border border-border">
              <div className="p-8 space-y-6 text-muted-foreground leading-relaxed">
                <p>
                  A transaction is flagged as a probable poisoning signal when <strong className="text-foreground">2 or more gates fail</strong>.
                </p>
                <p>
                  Each gate runs independently. Results are logged as pass, fail, or unknown (see below).
                  One failed gate alone usually isn't enough—poisoning patterns show multiple red flags together.
                </p>
                <div className="bg-muted/30 border border-border p-6 text-sm font-mono mt-6">
                  <div className="mb-2 text-foreground">Simple version:</div>
                  if (failed_gates &gt;= 2) → FLAG as probable signal<br />
                  else → PASS (no flag)
                </div>
                <p className="text-sm">
                  Flagged doesn't mean confirmed scam. It means "this pattern looks suspicious, review the evidence."
                </p>
              </div>
            </div>
          </section>

          {/* Uncertainty Handling */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">Handling Missing or Uncertain Data</h2>
            <div className="space-y-6 text-muted-foreground leading-relaxed">
              <p>
                Sometimes a gate can't evaluate properly—maybe the data source is slow, or historical context isn't available yet.
                When this happens, PoisonTrace uses <strong className="text-foreground">fail-closed behavior</strong>.
              </p>
              <div className="border border-border">
                <div className="border-b border-border p-6 bg-muted/30">
                  <h3>Unknown Gate Policy</h3>
                </div>
                <div className="p-6 space-y-4">
                  <div className="text-sm">
                    <strong className="text-foreground">Term: Fail-closed</strong>
                    <p className="text-muted-foreground mt-1">
                      If required data is unknown, candidate emission is blocked and the reason is logged.
                    </p>
                  </div>
                  <div className="text-sm">
                    <strong className="text-foreground">What happens:</strong>
                  </div>
                  <ul className="space-y-2 text-sm list-none">
                    <li className="before:content-['—'] before:mr-3 before:text-muted-foreground">
                      Transaction stays in queue (not flagged, not cleared)
                    </li>
                    <li className="before:content-['—'] before:mr-3 before:text-muted-foreground">
                      Reason logged explicitly ("Gate 2: unknown - data unavailable")
                    </li>
                    <li className="before:content-['—'] before:mr-3 before:text-muted-foreground">
                      Interface shows "unknown-gate" state
                    </li>
                    <li className="before:content-['—'] before:mr-3 before:text-muted-foreground">
                      Re-evaluated when data becomes available
                    </li>
                  </ul>
                </div>
              </div>
              <p className="text-sm font-mono">
                <span className="text-foreground">Example:</span> If counterparty history can't load due to indexing delays,
                Gate 3 returns "unknown" and the transaction isn't flagged until we can verify the data.
              </p>
            </div>
          </section>

          {/* Limitations */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">What PoisonTrace Does (and Doesn't) Claim</h2>
            <div className="border border-border divide-y divide-border">
              <div className="p-8">
                <h3 className="mb-3">What "Flagged" Means</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  A flag means probable poisoning patterns were detected based on the rules above.
                  It does NOT mean the transaction is definitely a scam, that specific victims have been identified,
                  or that action must be taken. Think of it as "this pattern looks suspicious, here's the evidence."
                </p>
              </div>
              <div className="p-8">
                <h3 className="mb-3">What We Don't Claim</h3>
                <ul className="text-sm space-y-2 list-none text-muted-foreground">
                  <li className="before:content-['—'] before:mr-3">Zero false positives (some legitimate activity may match patterns)</li>
                  <li className="before:content-['—'] before:mr-3">Scam confirmation (we detect patterns, not intent)</li>
                  <li className="before:content-['—'] before:mr-3">Victim attribution (we don't identify specific targets)</li>
                  <li className="before:content-['—'] before:mr-3">Prediction of future attacks (we detect, not predict)</li>
                  <li className="before:content-['—'] before:mr-3">Coverage of all attack types (unknown tactics won't match existing rules)</li>
                </ul>
              </div>
              <div className="p-8">
                <h3 className="mb-3">Data Dependencies</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  Detection quality depends on Solana data source uptime, transaction indexing speed, and dataset completeness.
                  Gaps in chain data can delay detection or produce unknown states.
                </p>
              </div>
            </div>
          </section>

          {/* Rule Updates */}
          <section>
            <h2 className="text-xl mb-6 tracking-tight">How Rules Get Updated</h2>
            <p className="text-muted-foreground leading-relaxed mb-6">
              As new poisoning tactics emerge, detection rules and thresholds are adjusted.
              All changes are versioned, logged, and documented. Updates are communicated via the interface
              when methodology changes roll out.
            </p>
            <div className="border border-border p-6 bg-muted/30">
              <div className="text-sm font-mono mb-2 text-muted-foreground">Current methodology version:</div>
              <div className="flex items-baseline gap-3">
                <span className="text-foreground font-mono">v1.0.0</span>
                <span className="text-xs text-muted-foreground font-mono">Released May 2026</span>
              </div>
            </div>
          </section>
        </div>

        <div className="mt-20 pt-8 border-t border-border">
          <Link to="/" className="text-sm text-muted-foreground hover:text-foreground transition-colors">
            ← Back
          </Link>
        </div>
      </div>
    </div>
  );
}
