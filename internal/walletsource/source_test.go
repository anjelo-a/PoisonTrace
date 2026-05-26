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
