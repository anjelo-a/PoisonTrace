import { Link } from "react-router";

export default function Landing() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-border bg-background">
        <div className="max-w-7xl mx-auto px-8 py-6 flex items-center justify-between">
          <div className="font-mono tracking-tight">PoisonTrace</div>
          <nav className="flex items-center gap-12">
            <Link to="/methodology" className="text-muted-foreground hover:text-foreground transition-colors text-sm">
              Methodology
            </Link>
            <Link to="/app/candidates" className="text-foreground text-sm hover:underline underline-offset-4">
              Review Patterns
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <section className="max-w-4xl mx-auto px-8 pt-24 pb-16">
        <div className="mb-6 text-xs uppercase tracking-widest text-muted-foreground font-mono">
          Solana Poisoning Detection
        </div>
        <h1 className="text-5xl mb-6 leading-tight tracking-tight">
          Scams Show Patterns.<br />
          PoisonTrace Surfaces Them.
        </h1>
        <p className="text-lg text-muted-foreground mb-8 max-w-2xl leading-relaxed">
          PoisonTrace scans bounded Solana transaction windows for probable poisoning signals—lookalike addresses,
          dust attacks, repeat injections. Rule-based detection. Transparent reasoning. Fail-closed by default.
        </p>

        {/* Live metric - understated */}
        <div className="mb-12 pb-8 border-b border-border/50">
          <div className="flex items-baseline gap-3">
            <div className="text-3xl font-mono tracking-tight">847</div>
            <div className="text-sm text-muted-foreground">probable poisoning signals detected this week</div>
          </div>
          <div className="text-xs text-muted-foreground font-mono mt-2">
            Based on current scan window and configured rules.
          </div>
        </div>

        {/* CTAs */}
        <div className="flex gap-6 mb-8">
          <Link
            to="/app/candidates"
            className="px-8 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors"
          >
            Review Poisoning Patterns
          </Link>
          <Link
            to="/methodology"
            className="px-8 py-3 border border-border hover:border-foreground transition-colors"
          >
            View Methodology
          </Link>
        </div>

        {/* Trust disclaimer */}
        <p className="text-sm text-muted-foreground mb-10 leading-relaxed max-w-lg">
          PoisonTrace flags patterns for review. You make the final decision.
        </p>

        {/* Proof strip - understated */}
        <div className="flex gap-8 text-xs text-muted-foreground font-mono">
          <div className="flex items-center gap-2">
            <div className="w-1 h-1 bg-muted-foreground" />
            <span>Rule-based only</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-1 h-1 bg-muted-foreground" />
            <span>Fail-closed by default</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-1 h-1 bg-muted-foreground" />
            <span>No black-box scoring</span>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="border-y border-border py-20">
        <div className="max-w-5xl mx-auto px-8">
          <h2 className="text-2xl mb-12 tracking-tight">How It Works</h2>
          <div className="relative">
            {/* Timeline connector line */}
            <div className="absolute left-[30px] top-0 bottom-0 w-px bg-border"></div>

            <div className="space-y-0">
              {/* Step 1 */}
              <div className="relative pl-20 pb-12">
                <div className="absolute left-[22px] top-2 w-4 h-4 border-2 border-foreground bg-background"></div>
                <div className="text-xs font-mono text-muted-foreground mb-2">01</div>
                <h3 className="mb-3">Ingest Transaction Windows</h3>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  PoisonTrace ingests bounded windows of Solana transaction data from public chain sources.
                  No wallet connection required—scans are performed on observable on-chain activity.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Evidence: Transaction dataset metadata showing block range, timestamp bounds, and record count
                </p>
              </div>

              {/* Step 2 */}
              <div className="relative pl-20 pb-12">
                <div className="absolute left-[22px] top-2 w-4 h-4 border-2 border-foreground bg-background"></div>
                <div className="text-xs font-mono text-muted-foreground mb-2">02</div>
                <h3 className="mb-3">Apply Poisoning Rules</h3>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  Each transaction is evaluated against explicit poisoning detection rules: lookalike address patterns,
                  dust amount thresholds, rapid-fire timing, first-time counterparty signals.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Evidence: Per-transaction gate log showing pass/fail/unknown status for each rule
                </p>
              </div>

              {/* Step 3 */}
              <div className="relative pl-20 pb-12">
                <div className="absolute left-[22px] top-2 w-4 h-4 border-2 border-foreground bg-background"></div>
                <div className="text-xs font-mono text-muted-foreground mb-2">03</div>
                <h3 className="mb-3">Surface Probable Signals</h3>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  When multiple gates fail, the transaction is flagged as a probable poisoning signal.
                  You see which rules failed, observed values vs expected thresholds, and gate-by-gate reasoning.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Evidence: Flagged transaction report with observed/expected values and failure trace
                </p>
              </div>

              {/* Step 4 */}
              <div className="relative pl-20">
                <div className="absolute left-[22px] top-2 w-4 h-4 border-2 border-foreground bg-background"></div>
                <div className="text-xs font-mono text-muted-foreground mb-2">04</div>
                <h3 className="mb-3">Export Evidence Artifacts</h3>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  Download forensic reports containing full detection logs, timestamps, and gate traces.
                  Use these artifacts for your own analysis or share with investigators if needed.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Evidence: PDF/CSV/JSON exports with complete audit trail and detection metadata
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Scan Window Summary */}
      <section className="py-16">
        <div className="max-w-5xl mx-auto px-8">
          <h2 className="text-lg mb-8 tracking-tight">Scan Window Summary (24h)</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-12">
            <div>
              <div className="text-3xl font-mono tracking-tight mb-2">7</div>
              <div className="text-xs text-muted-foreground uppercase tracking-widest">Candidates Emitted</div>
            </div>
            <div>
              <div className="text-3xl font-mono tracking-tight mb-2">1,236</div>
              <div className="text-xs text-muted-foreground uppercase tracking-widest">Passed Gates</div>
            </div>
            <div>
              <div className="text-3xl font-mono tracking-tight mb-2">3</div>
              <div className="text-xs text-muted-foreground uppercase tracking-widest">Unknown-Gate Blocked</div>
            </div>
            <div>
              <div className="text-3xl font-mono tracking-tight mb-2">99.4%</div>
              <div className="text-xs text-muted-foreground uppercase tracking-widest">Pass Rate</div>
            </div>
          </div>
        </div>
      </section>

      {/* Sample Explanation Snapshot */}
      <section className="py-20">
        <div className="max-w-5xl mx-auto px-8">
          <h2 className="text-2xl mb-6 tracking-tight">Sample Explanation Snapshot</h2>
          <p className="text-sm text-muted-foreground mb-8 font-mono">
            Illustrative example, not production thresholds.
          </p>
          <div className="border border-border">
            <div className="border-b border-border p-6 bg-muted/30">
              <div className="text-xs font-mono text-muted-foreground mb-1">Transaction Signature</div>
              <div className="font-mono text-sm">5KqR...d8Hs</div>
            </div>
            <div className="divide-y divide-border">
              <div className="p-6">
                <div className="grid grid-cols-[200px_1fr_100px] gap-4 text-sm font-mono mb-2">
                  <div className="text-muted-foreground">Gate: Lookalike Similarity</div>
                  <div>3 char diff (expected &gt;10)</div>
                  <div className="text-destructive-foreground">FAIL</div>
                </div>
              </div>
              <div className="p-6">
                <div className="grid grid-cols-[200px_1fr_100px] gap-4 text-sm font-mono mb-2">
                  <div className="text-muted-foreground">Gate: Min Repeat Injections</div>
                  <div>3 events (threshold &lt;2)</div>
                  <div className="text-destructive-foreground">FAIL</div>
                </div>
              </div>
              <div className="p-6">
                <div className="grid grid-cols-[200px_1fr_100px] gap-4 text-sm font-mono mb-2">
                  <div className="text-muted-foreground">Gate: Recency Window</div>
                  <div>18h ago (&lt;24h active)</div>
                  <div className="text-muted-foreground">PASS</div>
                </div>
              </div>
              <div className="p-6 bg-muted/30">
                <div className="grid grid-cols-[200px_1fr_100px] gap-4 text-sm font-mono mb-2">
                  <div className="text-muted-foreground">Gate: Baseline Required</div>
                  <div>Counterparty history unavailable</div>
                  <div className="text-muted-foreground">UNKNOWN</div>
                </div>
              </div>
            </div>
            <div className="border-t border-border p-6 bg-muted/30 text-sm font-mono">
              2 gates failed, 1 unknown → Candidate emission blocked pending data availability
            </div>
          </div>
        </div>
      </section>

      {/* Trust Section - asymmetric spacing (shorter gap before) */}
      <section className="py-16">
        <div className="max-w-5xl mx-auto px-8">
          <h2 className="text-2xl mb-12 tracking-tight">Built on Explicit Rules</h2>
          <div className="space-y-0">
            {/* Item 1 */}
            <div className="grid grid-cols-[60px_1fr] gap-6 items-start py-8 border-b border-border">
              <div className="text-sm font-mono text-muted-foreground pt-1">01</div>
              <div>
                <div className="flex items-center gap-3 mb-3">
                  <h3>Transparent Reasoning</h3>
                  <span className="text-xs font-mono text-muted-foreground border border-border px-2 py-1">VERIFIABLE</span>
                </div>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  Every detection includes gate-by-gate outcomes, observed values, expected thresholds, and linked evidence.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Verify: Open any flagged transaction and inspect the gate trace and observed-vs-expected values.
                </p>
              </div>
            </div>

            {/* Item 2 */}
            <div className="grid grid-cols-[60px_1fr] gap-6 items-start py-8 border-b border-border">
              <div className="text-sm font-mono text-muted-foreground pt-1">02</div>
              <div>
                <div className="flex items-center gap-3 mb-3">
                  <h3>Rule-Based Detection</h3>
                  <span className="text-xs font-mono text-muted-foreground border border-border px-2 py-1">NO ML</span>
                </div>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  Signals come from explicit rules and thresholds, not predictive black-box scoring.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Verify: Review rule names, thresholds, and pass/fail outcomes in the detection log.
                </p>
              </div>
            </div>

            {/* Item 3 */}
            <div className="grid grid-cols-[60px_1fr] gap-6 items-start py-8">
              <div className="text-sm font-mono text-muted-foreground pt-1">03</div>
              <div>
                <div className="flex items-center gap-3 mb-3">
                  <h3>Fail-Closed Behavior</h3>
                  <span className="text-xs font-mono text-muted-foreground border border-border px-2 py-1">FAIL-CLOSED</span>
                </div>
                <p className="text-muted-foreground leading-relaxed mb-3">
                  If required data is unknown, candidate emission is blocked and the reason is logged.
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  Verify: Check unknown_gate_reason and incomplete_window markers in blocked outcomes.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Footer - tighter spacing */}
      <footer className="border-t border-border py-12">
        <div className="max-w-5xl mx-auto px-8">
          <div className="grid grid-cols-3 gap-16 mb-12">
            <div>
              <h4 className="mb-4 text-sm">Product</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="hover:text-foreground transition-colors cursor-pointer">Documentation</div>
                <div className="hover:text-foreground transition-colors cursor-pointer">Methodology</div>
                <div className="hover:text-foreground transition-colors cursor-pointer">API Access</div>
              </div>
            </div>
            <div>
              <h4 className="mb-4 text-sm">Legal</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="hover:text-foreground transition-colors cursor-pointer">Privacy Policy</div>
                <div className="hover:text-foreground transition-colors cursor-pointer">Terms of Service</div>
                <div className="hover:text-foreground transition-colors cursor-pointer">Data Handling</div>
              </div>
            </div>
            <div>
              <h4 className="mb-4 text-sm">Support</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="hover:text-foreground transition-colors cursor-pointer">Help Center</div>
                <div className="hover:text-foreground transition-colors cursor-pointer">Report Issue</div>
              </div>
            </div>
          </div>
          <div className="text-xs text-muted-foreground font-mono border-t border-border pt-6">
            © 2026 PoisonTrace
          </div>
        </div>
      </footer>
    </div>
  );
}
