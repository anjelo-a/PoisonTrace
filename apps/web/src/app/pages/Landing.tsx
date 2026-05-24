import { Link } from "react-router";
import "./Landing.css";

const proofItems = ["Rule-based only", "Fail-closed by default", "No black-box scoring"];

const poisoningSequence = [
  {
    title: "How It Starts",
    text: "You receive a tiny or zero-value transfer from an unfamiliar address that looks similar to one you already trust.",
  },
  {
    title: "How People Get Tricked",
    text: "During a later transfer, someone copies an address from history and accidentally pastes the lookalike.",
  },
  {
    title: "Why It Works",
    text: "Wallet addresses are long and visually dense, so minor character differences are easy to miss at a glance.",
  },
];

const technicalRows = [
  ["1", "Read bounded history", "Ingest a wallet scan window and the baseline needed to compare counterparties."],
  ["2", "Apply explicit gates", "Check lookalike similarity, dust behavior, recency, repeats, and baseline completeness."],
  ["3", "Block uncertainty", "If a required gate is unknown, candidate emission stops and the reason is preserved."],
];

const steps = [
  {
    title: "Ingest Transaction Windows",
    text: "PoisonTrace ingests bounded windows of Solana transaction data from public chain sources. No wallet connection required—scans are performed on observable on-chain activity.",
    evidence: "Evidence: Transaction dataset metadata showing block range, timestamp bounds, and record count",
  },
  {
    title: "Apply Poisoning Rules",
    text: "Each transaction is evaluated against explicit poisoning detection rules: lookalike address patterns, dust amount thresholds, rapid-fire timing, first-time counterparty signals.",
    evidence: "Evidence: Per-transaction gate log showing pass/fail/unknown status for each rule",
  },
  {
    title: "Surface Probable Signals",
    text: "When multiple gates fail, the transaction is flagged as a probable poisoning signal. You see which rules failed, observed values vs expected thresholds, and gate-by-gate reasoning.",
    evidence: "Evidence: Flagged transaction report with observed/expected values and failure trace",
  },
  {
    title: "Export Evidence Artifacts",
    text: "Download forensic reports containing full detection logs, timestamps, and gate traces. Use these artifacts for your own analysis or share with investigators if needed.",
    evidence: "Evidence: PDF/CSV/JSON exports with complete audit trail and detection metadata",
  },
];

const summaryStats = [
  ["7", "Candidates Emitted"],
  ["1,236", "Passed Gates"],
  ["3", "Unknown-Gate Blocked"],
  ["99.4%", "Pass Rate"],
];

const checklist = [
  "Do not trust an address only because it appears in recent history.",
  "Verify the full destination address against a saved contact or another trusted source.",
  "Treat tiny inbound transfers from new counterparties as potential bait.",
  "For high-value transfers, send a small test amount first whenever possible.",
];

const explanationRows = [
  ["Gate: Lookalike Similarity", "3 char diff (expected >10)", "FAIL"],
  ["Gate: Min Repeat Injections", "3 events (threshold <2)", "FAIL"],
  ["Gate: Recency Window", "18h ago (<24h active)", "PASS"],
  ["Gate: Baseline Required", "Counterparty history unavailable", "UNKNOWN"],
];

const ruleItems = [
  {
    label: "VERIFIABLE",
    title: "Transparent Reasoning",
    text: "Every detection includes gate-by-gate outcomes, observed values, expected thresholds, and linked evidence.",
    verify: "Verify: Open any flagged transaction and inspect the gate trace and observed-vs-expected values.",
  },
  {
    label: "NO ML",
    title: "Rule-Based Detection",
    text: "Signals come from explicit rules and thresholds, not predictive black-box scoring.",
    verify: "Verify: Review rule names, thresholds, and pass/fail outcomes in the detection log.",
  },
  {
    label: "FAIL-CLOSED",
    title: "Fail-Closed Behavior",
    text: "If required data is unknown, candidate emission is blocked and the reason is logged.",
    verify: "Verify: Check unknown_gate_reason and incomplete_window markers in blocked outcomes.",
  },
];

export default function Landing() {
  return (
    <div className="landing-shell min-h-screen">
      <header className="landing-header">
        <div className="landing-container flex items-center justify-between gap-6 py-5">
          <Link to="/" className="font-mono text-sm tracking-tight text-white">
            PoisonTrace
          </Link>
          <nav className="flex items-center gap-5 text-[11px] sm:gap-8">
            <Link to="/methodology" className="landing-nav-link">
              Methodology
            </Link>
            <Link to="/app/candidates" className="landing-nav-link">
              Review Patterns
            </Link>
          </nav>
        </div>
      </header>

      <main>
        <section className="landing-hero landing-section">
          <div className="landing-container landing-hero-grid">
            <div className="landing-hero-copy">
              <div className="landing-kicker">Solana Poisoning Detection</div>
              <h1 className="landing-display">
                Scams Show Patterns.<br />
                PoisonTrace Surfaces Them.
              </h1>
              <p className="landing-lede">
                Rule-based Solana scans for lookalike addresses, dust attacks, and repeat injections.
              </p>

              <div className="landing-metric">
                <div className="landing-metric-value">847</div>
                <div className="landing-metric-line">probable poisoning signals detected this week</div>
                <div className="landing-metric-line">based on current scan windows and configured rules</div>
              </div>

              <div className="flex flex-col gap-3 sm:flex-row">
                <Link to="/app/candidates" className="landing-button landing-button-primary">
                  Review Poisoning Patterns
                </Link>
                <Link to="/methodology" className="landing-button landing-button-secondary">
                  View Methodology
                </Link>
              </div>

              <p className="landing-hero-note">
                PoisonTrace flags patterns for review. You make the final decision.
              </p>

              <div className="landing-proof-strip">
                {proofItems.map((item) => (
                  <div key={item} className="flex items-center gap-2">
                    <span className="landing-dot" />
                    <span>{item}</span>
                  </div>
                ))}
              </div>
            </div>

            <figure className="landing-stitch-hero" aria-hidden="true">
              <img src="/stitch/generated/hero-prism.png" alt="" loading="eager" />
            </figure>
          </div>
        </section>

        <section className="landing-section landing-band">
          <div className="landing-container landing-split">
            <div>
              <div className="landing-kicker">Threat Pattern</div>
              <h2 className="landing-heading">Wallet Poisoning</h2>
              <p className="landing-copy">
                Wallet poisoning is a copy-and-paste scam. Attackers send tiny transfers from lookalike addresses so those
                addresses appear in your recent activity. Later, they hope you copy the wrong address during a real transfer.
              </p>
            </div>
            <div className="landing-thread-list">
              {poisoningSequence.map((item, index) => (
                <article key={item.title} className="landing-thread-item">
                  <div className="landing-thread-index">{String(index + 1).padStart(2, "0")}</div>
                  <div>
                    <h3>{item.title}</h3>
                    <p>{item.text}</p>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="landing-section">
          <div className="landing-container landing-feature-split landing-visual-led">
            <figure className="landing-stitch-optics" aria-hidden="true">
              <img src="/stitch/generated/forensic-optics.png" alt="" loading="lazy" />
            </figure>
            <div className="landing-feature-copy">
              <div className="landing-kicker">Technical Core</div>
              <h2 className="landing-heading">What PoisonTrace Actually Does</h2>
              <p className="landing-copy">
                PoisonTrace is a scanner-first Solana analysis tool. It does not connect to wallets or execute
                transactions. It turns on-chain history into reviewable, auditable poisoning candidates.
              </p>
              <div className="landing-process-list">
                {technicalRows.map(([index, title, text]) => (
                  <div key={title} className="landing-process-row">
                    <span>{index}</span>
                    <div>
                      <h3>{title}</h3>
                      <p>{text}</p>
                    </div>
                  </div>
                ))}
              </div>
              <Link to="/methodology" className="landing-text-link">
                View the full methodology
              </Link>
            </div>
          </div>
        </section>

        <section className="landing-section landing-band">
          <div className="landing-container">
            <div className="landing-section-head">
              <div>
                <div className="landing-kicker">Pipeline</div>
              <h2 className="landing-heading">How It Works</h2>
            </div>
            </div>
            <div className="landing-timeline">
              {steps.map((step, index) => (
                <article key={step.title} className="landing-step">
                  <div className="landing-step-index">{String(index + 1).padStart(2, "0")}</div>
                  <div>
                    <h3>{step.title}</h3>
                    <p>{step.text}</p>
                    <div className="landing-evidence">{step.evidence}</div>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="landing-section">
          <div className="landing-container landing-feature-split landing-visual-led">
            <figure className="landing-stitch-ledger-detail" aria-hidden="true">
              <img src="/stitch/generated/evidence-ledger.png" alt="" loading="lazy" />
            </figure>
            <div className="landing-feature-copy">
              <div className="landing-chapter-head landing-chapter-head-stacked">
                <div className="landing-kicker">Evidence Ledger</div>
                <h2 className="landing-heading">A Readout You Can Audit</h2>
              <span className="landing-status">resolved</span>
            </div>
            <p className="landing-section-intro">
              Each scan returns a bounded ledger of outcomes: what passed, what was blocked, and what still needs
              evidence. The numbers summarize the run without asking you to trust a black-box score.
            </p>
              <div className="landing-stat-stack">
                {summaryStats.map(([value, label]) => (
                  <div key={label} className="landing-stat-item">
                    <div className="font-mono text-3xl tracking-tight text-white">{value}</div>
                    <div className="mt-3 text-[11px] uppercase tracking-[0.18em] text-white/48">{label}</div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="landing-section landing-band">
          <div className="landing-container">
            <div className="landing-chapter-head">
              <div>
                <div className="landing-kicker">Candidate Evidence</div>
                <h2 className="landing-heading">Sample Explanation Snapshot</h2>
              </div>
            </div>
            <p className="landing-section-intro font-mono text-sm text-white/50">
              Illustrative example, not production thresholds.
            </p>
            <div className="landing-explanation">
              <div className="landing-explanation-head">
                <div className="mb-1 font-mono text-[11px] uppercase tracking-[0.18em] text-white/42">
                  Transaction Signature
                </div>
                <div className="font-mono text-sm text-white">5KqR...d8Hs</div>
              </div>
              {explanationRows.map(([gate, value, status]) => (
                <div key={gate} className="landing-explanation-row">
                  <span>{gate}</span>
                  <span>{value}</span>
                  <span className={status === "FAIL" ? "text-red-200" : "text-white/54"}>{status}</span>
                </div>
              ))}
              <div className="landing-explanation-result">
                2 gates failed, 1 unknown → Candidate emission blocked pending data availability
              </div>
            </div>
          </div>
        </section>

        <section className="landing-section">
          <div className="landing-container">
            <div className="landing-chapter-head">
              <div>
                <div className="landing-kicker">Operational Guardrails</div>
                <h2 className="landing-heading">Quick Safety Checklist</h2>
              </div>
            </div>
            <p className="landing-section-intro">
              Human review stays separate from scanner output. Use this short checklist before trusting any address copied
              from recent history.
            </p>
            <div className="landing-checklist">
              {checklist.map((item, index) => (
                <div key={item} className="landing-check-row">
                  <span>{String(index + 1).padStart(2, "0")}</span>
                  <p>{item}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="landing-section landing-band">
          <div className="landing-container landing-feature-split landing-visual-led-copy-first">
            <div className="landing-feature-copy">
              <div className="landing-chapter-head landing-chapter-head-stacked">
                <div className="landing-kicker">Signal Truth</div>
                <h2 className="landing-heading">Built on Explicit Rules</h2>
              </div>
            <p className="landing-section-intro">
              These are the scanner principles behind the evidence flow. They explain how PoisonTrace limits ambiguity
              before anything becomes a probable candidate.
            </p>
              <div className="landing-rule-list">
                {ruleItems.map((item, index) => (
                  <article key={item.title} className="landing-rule-item">
                    <div className="landing-step-index">{String(index + 1).padStart(2, "0")}</div>
                    <div>
                      <div className="mb-3 flex flex-wrap items-center gap-3">
                        <h3>{item.title}</h3>
                        <span>{item.label}</span>
                      </div>
                      <p>{item.text}</p>
                      <div className="landing-evidence">{item.verify}</div>
                    </div>
                  </article>
                ))}
              </div>
            </div>
            <figure className="landing-stitch-crystal" aria-hidden="true">
              <img src="/stitch/generated/methodology-diamond.png" alt="" loading="lazy" />
            </figure>
          </div>
        </section>
      </main>

      <footer className="landing-footer">
        <div className="landing-container">
          <div className="grid gap-10 sm:grid-cols-3">
            <FooterColumn title="Product" items={["Documentation", "Methodology", "API Access"]} />
            <FooterColumn title="Legal" items={["Privacy Policy", "Terms of Service", "Data Handling"]} />
            <FooterColumn title="Support" items={["Help Center", "Report Issue"]} />
          </div>
          <div className="mt-10 border-t border-white/10 pt-6 font-mono text-xs text-white/42">© 2026 PoisonTrace</div>
        </div>
      </footer>
    </div>
  );
}

function FooterColumn({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <h4 className="mb-4 text-sm text-white">{title}</h4>
      <div className="space-y-2 text-sm text-white/48">
        {items.map((item) => (
          <div key={item} className="transition-colors hover:text-white">
            {item}
          </div>
        ))}
      </div>
    </div>
  );
}
