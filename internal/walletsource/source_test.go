package walletsource

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"poisontrace/internal/helius"
)

type fakeHeliusClient struct {
	pages map[string][]helius.EnhancedPage
}

func (f fakeHeliusClient) FetchEnhancedPage(_ context.Context, walletAddress string, before string) (helius.EnhancedPage, error) {
	pages := f.pages[walletAddress]
	if len(pages) == 0 {
		return helius.EnhancedPage{}, nil
	}
	if before == "" {
		return pages[0], nil
	}
	for i, p := range pages {
		if p.Before == before && i+1 < len(pages) {
			return pages[i+1], nil
		}
	}
	return helius.EnhancedPage{}, nil
}

func TestSourceAcceptsBoringOutboundWallet(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-candidate", 100, "Seed111111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "1000"),
			},
			Before: "seed-page-1",
		}},
		"Candidate111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("candidate-out", 101, "Candidate111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "2000"),
				nativeTx("candidate-in", 102, "Friend111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "3000"),
			},
			Before: "candidate-page-1",
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:   seedPath,
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected one accepted wallet, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address != "Candidate111111111111111111111111111111111" {
		t.Fatalf("unexpected accepted wallet %s", res.Accepted[0].Address)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != "Candidate111111111111111111111111111111111" {
		t.Fatalf("unexpected accepted file %q", string(raw))
	}
}

func TestSourceRejectsCapAndBurstWallets(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-cap", 100, "Seed111111111111111111111111111111111111111", "Cap11111111111111111111111111111111111111", "1000"),
				nativeTx("seed-to-burst", 101, "Seed111111111111111111111111111111111111111", "Burst111111111111111111111111111111111111", "1000"),
			},
			Before: "seed-page-1",
		}},
		"Cap11111111111111111111111111111111111111": {
			{Transactions: []helius.EnhancedTransaction{nativeTx("cap-1", 101, "Cap11111111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "1")}, Before: "cap-page-1"},
			{Transactions: []helius.EnhancedTransaction{nativeTx("cap-2", 102, "Cap11111111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "1")}, Before: "cap-page-2"},
		},
		"Burst111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("burst-1", 110, "Burst111111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "1"),
				nativeTx("burst-2", 110, "Burst111111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "1"),
				nativeTx("burst-3", 110, "Burst111111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "1"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:     seedPath,
		OutPath:            outPath,
		RejectedOutPath:    rejectedPath,
		ScanStart:          time.Unix(90, 0),
		ScanEnd:            time.Unix(200, 0),
		BaselineLookback:   60 * time.Second,
		TargetCount:        10,
		CandidateMaxPages:  2,
		MaxSameTimestampTX: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 {
		t.Fatalf("expected no accepted wallets, got %d", len(res.Accepted))
	}

	rejections := map[string]string{}
	for _, r := range res.Rejected {
		rejections[r.Address] = r.Reason
	}
	if !strings.HasPrefix(rejections["Cap11111111111111111111111111111111111111"], "activity_cap_reached:") {
		t.Fatalf("expected cap rejection, got %q", rejections["Cap11111111111111111111111111111111111111"])
	}
	if rejections["Burst111111111111111111111111111111111111"] != "bursty_same_timestamp_activity" {
		t.Fatalf("expected burst rejection, got %q", rejections["Burst111111111111111111111111111111111111"])
	}
}

func TestSourceDiscoveredScoresSeedWalletsDirectly(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Scraped11111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("scraped-out", 101, "Scraped11111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "2000"),
			},
		}},
	}}

	res, err := SourceDiscovered(context.Background(), client, []string{"Scraped11111111111111111111111111111111111"}, Options{
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected scraped seed to be accepted, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address != "Scraped11111111111111111111111111111111111" {
		t.Fatalf("unexpected accepted wallet %s", res.Accepted[0].Address)
	}
}

func TestSourceDiscoveredDoesNotScoreNeighbors(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Scraped11111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("scraped-to-neighbor", 101, "Scraped11111111111111111111111111111111111", "Neighbor111111111111111111111111111111111", "2000"),
			},
		}},
		"Neighbor111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("neighbor-out", 102, "Neighbor111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "2000"),
			},
		}},
	}}

	res, err := SourceDiscovered(context.Background(), client, []string{"Scraped11111111111111111111111111111111111"}, Options{
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected only scraped seed to be accepted, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address == "Neighbor111111111111111111111111111111111" {
		t.Fatal("did not expect discovered neighbor to be scored in scraped mode")
	}
}

func TestSourceRejectsUnknownDustSPLActivity(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-candidate", 100, "Seed111111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "1000"),
			},
		}},
		"Candidate111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				tokenTx("candidate-unknown-spl", 101, "Candidate111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "UnknownMint111111111111111111111111111111", "42"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:   seedPath,
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 {
		t.Fatalf("expected unknown-dust SPL wallet to be rejected, got %d accepted", len(res.Accepted))
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("expected one rejected wallet, got %d", len(res.Rejected))
	}
	if res.Rejected[0].Reason != "unknown_dust_spl_activity" {
		t.Fatalf("expected unknown dust rejection, got %q", res.Rejected[0].Reason)
	}
}

func TestSourceAcceptsKnownDustSPLActivity(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	const usdc = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-candidate", 100, "Seed111111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "1000"),
			},
		}},
		"Candidate111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				tokenTx("candidate-known-spl", 101, "Candidate111111111111111111111111111111111", "Friend111111111111111111111111111111111111", usdc, "42"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:   seedPath,
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected known-dust SPL wallet to be accepted, got %d accepted", len(res.Accepted))
	}
}

func TestSourceAttackerModeRequiresOutboundDust(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-attacker", 100, "Seed111111111111111111111111111111111111111", "Attacker111111111111111111111111111111111", "5000"),
			},
		}},
		"Attacker111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("attacker-out-1", 101, "Attacker111111111111111111111111111111111", "Victim11111111111111111111111111111111111", "1"),
				nativeTx("attacker-out-2", 102, "Attacker111111111111111111111111111111111", "Victim22222222222222222222222222222222222", "1"),
			},
		}},
		"Bystander111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("bystander-out", 101, "Bystander111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "5000"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:          seedPath,
		SeedWallets:             []string{"Bystander111111111111111111111111111111111"},
		ScoreSeedWallets:        true,
		DiscoverNeighbors:       true,
		OutPath:                 outPath,
		RejectedOutPath:         rejectedPath,
		ScanStart:               time.Unix(90, 0),
		ScanEnd:                 time.Unix(200, 0),
		BaselineLookback:        60 * time.Second,
		TargetCount:             10,
		MinScanInboundDust:      1,
		MinUniqueDustRecipients: 2,
		SourceMode:              SourceModeAttackerOutboundDust,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected one accepted wallet in attacker mode, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address != "Attacker111111111111111111111111111111111" {
		t.Fatalf("unexpected accepted wallet %s", res.Accepted[0].Address)
	}
	reasons := map[string]string{}
	for _, r := range res.Rejected {
		reasons[r.Address] = r.Reason
	}
	if reasons["Bystander111111111111111111111111111111111"] != "insufficient_outbound_dust_activity" {
		t.Fatalf("expected outbound dust rejection for bystander, got %q", reasons["Bystander111111111111111111111111111111111"])
	}
}

func TestSourceAttackerModeCanRequireUniqueDustRecipients(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-attacker", 100, "Seed111111111111111111111111111111111111111", "Attacker111111111111111111111111111111111", "5000"),
			},
		}},
		"Attacker111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("attacker-out-1", 101, "Attacker111111111111111111111111111111111", "Victim11111111111111111111111111111111111", "1"),
				nativeTx("attacker-out-2", 102, "Attacker111111111111111111111111111111111", "Victim11111111111111111111111111111111111", "1"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:          seedPath,
		OutPath:                 outPath,
		RejectedOutPath:         rejectedPath,
		ScanStart:               time.Unix(90, 0),
		ScanEnd:                 time.Unix(200, 0),
		BaselineLookback:        60 * time.Second,
		TargetCount:             10,
		MinScanInboundDust:      1,
		MinUniqueDustRecipients: 2,
		SourceMode:              SourceModeAttackerOutboundDust,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 {
		t.Fatalf("expected no accepted wallets, got %d", len(res.Accepted))
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("expected one rejected wallet, got %d", len(res.Rejected))
	}
	if res.Rejected[0].Reason != "insufficient_unique_dust_recipients" {
		t.Fatalf("expected unique dust recipient rejection, got %q", res.Rejected[0].Reason)
	}
	if res.Rejected[0].UniqueDustRecipients != 1 {
		t.Fatalf("expected one unique dust recipient, got %d", res.Rejected[0].UniqueDustRecipients)
	}
}

func TestSourcePrioritizesPoisoningLikeWallets(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	poisonWallet := "PoisonWallet111111111111111111111111111111"
	boringWallet := "BoringWallet111111111111111111111111111111"
	legit := "LookalikeLegit1111111111111111111111ZZZZ"
	suspicious := "LookalikeBad11111111111111111111111ZZZZ"
	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-boring", 100, "Seed111111111111111111111111111111111111111", boringWallet, "1000"),
				nativeTx("seed-to-poison", 101, "Seed111111111111111111111111111111111111111", poisonWallet, "1000"),
			},
		}},
		boringWallet: {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("boring-out", 101, boringWallet, "Friend111111111111111111111111111111111111", "2000"),
				nativeTx("boring-in", 102, "Friend111111111111111111111111111111111111", boringWallet, "3000"),
			},
		}},
		poisonWallet: {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("poison-baseline-out", 80, poisonWallet, legit, "5000000"),
				nativeTx("poison-in-1", 101, suspicious, poisonWallet, "1"),
				nativeTx("poison-in-2", 102, suspicious, poisonWallet, "1"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:   seedPath,
		OutPath:          outPath,
		RejectedOutPath:  rejectedPath,
		ScanStart:        time.Unix(90, 0),
		ScanEnd:          time.Unix(200, 0),
		BaselineLookback: 60 * time.Second,
		TargetCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected one accepted wallet, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address != poisonWallet {
		t.Fatalf("expected poisoning-like wallet to be accepted first, got %s", res.Accepted[0].Address)
	}
	if res.Accepted[0].ScanInboundDustTransfers != 2 {
		t.Fatalf("expected two inbound dust transfers, got %d", res.Accepted[0].ScanInboundDustTransfers)
	}
	if res.Accepted[0].RepeatedInboundDustCounterparts != 1 {
		t.Fatalf("expected repeated inbound dust counterparty, got %d", res.Accepted[0].RepeatedInboundDustCounterparts)
	}
	if res.Accepted[0].LookalikeInboundDustMatches != 1 {
		t.Fatalf("expected lookalike inbound dust match, got %d", res.Accepted[0].LookalikeInboundDustMatches)
	}
}

func TestSourceCanRequireInboundDustActivity(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-candidate", 100, "Seed111111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "1000"),
			},
		}},
		"Candidate111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("candidate-out", 101, "Candidate111111111111111111111111111111111", "Friend111111111111111111111111111111111111", "2000"),
				nativeTx("candidate-in", 102, "Friend111111111111111111111111111111111111", "Candidate111111111111111111111111111111111", "3000"),
			},
		}},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:     seedPath,
		OutPath:            outPath,
		RejectedOutPath:    rejectedPath,
		ScanStart:          time.Unix(90, 0),
		ScanEnd:            time.Unix(200, 0),
		BaselineLookback:   60 * time.Second,
		TargetCount:        10,
		MinScanInboundDust: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 {
		t.Fatalf("expected no accepted wallets, got %d", len(res.Accepted))
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("expected one rejected wallet, got %d", len(res.Rejected))
	}
	if res.Rejected[0].Reason != "insufficient_inbound_dust_activity" {
		t.Fatalf("expected inbound dust rejection, got %q", res.Rejected[0].Reason)
	}
}

func TestSourceDeepDiveRecoversHighSignalCappedWallet(t *testing.T) {
	tmp := t.TempDir()
	seedPath := filepath.Join(tmp, "seeds.txt")
	outPath := filepath.Join(tmp, "accepted.txt")
	rejectedPath := filepath.Join(tmp, "rejected.tsv")
	if err := os.WriteFile(seedPath, []byte("Seed111111111111111111111111111111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wallet := "DeepDiveWallet11111111111111111111111111111"
	legit := "LookalikeLegit1111111111111111111111ZZZZ"
	suspicious := "LookalikeBad11111111111111111111111ZZZZ"

	client := fakeHeliusClient{pages: map[string][]helius.EnhancedPage{
		"Seed111111111111111111111111111111111111111": {{
			Transactions: []helius.EnhancedTransaction{
				nativeTx("seed-to-candidate", 100, "Seed111111111111111111111111111111111111111", wallet, "1000"),
			},
		}},
		wallet: {
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("baseline-out", 80, wallet, legit, "5000000"),
					nativeTx("inbound-dust-1", 101, suspicious, wallet, "1"),
					nativeTx("inbound-dust-2", 102, suspicious, wallet, "1"),
				},
				Before: "page-1",
			},
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("filler", 79, wallet, "Friend111111111111111111111111111111111111", "2000"),
				},
			},
		},
	}}

	res, err := Source(context.Background(), client, Options{
		SeedWalletFile:     seedPath,
		OutPath:            outPath,
		RejectedOutPath:    rejectedPath,
		ScanStart:          time.Unix(90, 0),
		ScanEnd:            time.Unix(200, 0),
		BaselineLookback:   60 * time.Second,
		TargetCount:        1,
		CandidateMaxPages:  1,
		MaxTXPerWallet:     50,
		MinScanInboundDust: 1,
		DeepDiveTopN:       1,
		DeepDiveMaxPages:   3,
		DeepDiveMaxTX:      200,
		DeepDiveMinScore:   40,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("expected deep dive to recover one accepted wallet, got %d", len(res.Accepted))
	}
	if res.Accepted[0].Address != wallet {
		t.Fatalf("unexpected accepted wallet %s", res.Accepted[0].Address)
	}
}

func nativeTx(signature string, ts int64, from string, to string, amount string) helius.EnhancedTransaction {
	return helius.EnhancedTransaction{
		Signature:     signature,
		Slot:          ts,
		TimestampUnix: ts,
		NativeTransfers: []helius.NativeTransfer{{
			FromUserAccount: from,
			ToUserAccount:   to,
			Amount:          amount,
		}},
	}
}

func tokenTx(signature string, ts int64, from string, to string, mint string, amount string) helius.EnhancedTransaction {
	decimals := 6
	return helius.EnhancedTransaction{
		Signature:     signature,
		Slot:          ts,
		TimestampUnix: ts,
		TokenTransfers: []helius.TokenTransfer{{
			FromUserAccount:  from,
			ToUserAccount:    to,
			FromTokenAccount: from + "Token",
			ToTokenAccount:   to + "Token",
			Mint:             mint,
			TokenAmount: helius.TokenAmount{
				Amount:   amount,
				Decimals: &decimals,
			},
			TokenStandard: "Fungible",
		}},
	}
}
