package walletsource

import (
	"context"
	"testing"
)

type fakeRPC struct {
	slot   int64
	blocks map[int64]RPCBlock
}

func (f fakeRPC) GetSlot(context.Context) (int64, error) {
	return f.slot, nil
}

func (f fakeRPC) GetBlock(_ context.Context, slot int64) (RPCBlock, error) {
	return f.blocks[slot], nil
}

func TestScrapeRecentWalletsKeepsQuietTransferSigners(t *testing.T) {
	rpc := fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {Transactions: []RPCBlockTxEnvelope{
				blockTx("Normal111111111111111111111111111111111111", "transfer"),
				blockTx("Swap11111111111111111111111111111111111111", "swap"),
			}},
		},
	}

	wallets, err := ScrapeRecentWallets(context.Background(), rpc, ScrapeOptions{
		BlockLookback: 1,
		MaxWallets:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 {
		t.Fatalf("expected one scraped wallet, got %d", len(wallets))
	}
	if wallets[0] != "Normal111111111111111111111111111111111111" {
		t.Fatalf("unexpected wallet %s", wallets[0])
	}
}

func TestScrapeRecentWalletsPrefersParsedTransferSource(t *testing.T) {
	rpc := fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {
				Transactions: []RPCBlockTxEnvelope{{
					Transaction: RPCTransaction{
						Message: RPCMessage{
							AccountKeys: []RPCAccountKey{{
								Pubkey:   "FeePayer1111111111111111111111111111111111",
								Signer:   true,
								Writable: true,
								Source:   "transaction",
							}},
							Instructions: []RPCInstruction{{
								Program: "system",
								Parsed: &RPCParsedInstruction{
									Type: "transfer",
									Info: map[string]any{
										"lamports": float64(100),
										"source":   "Source1111111111111111111111111111111111111",
									},
								},
							}},
						},
					},
				}},
			},
		},
	}

	wallets, err := ScrapeRecentWallets(context.Background(), rpc, ScrapeOptions{
		BlockLookback: 1,
		MaxWallets:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 || wallets[0] != "Source1111111111111111111111111111111111111" {
		t.Fatalf("expected parsed transfer source, got %#v", wallets)
	}
}

func TestScrapeRecentWalletsPrefersParsedTransferAuthority(t *testing.T) {
	wallets, err := ScrapeRecentWallets(context.Background(), fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {
				Transactions: []RPCBlockTxEnvelope{{
					Transaction: RPCTransaction{
						Message: RPCMessage{
							AccountKeys: []RPCAccountKey{{
								Pubkey:   "FeePayer1111111111111111111111111111111111",
								Signer:   true,
								Writable: true,
								Source:   "transaction",
							}},
							Instructions: []RPCInstruction{{
								Program: "spl-token",
								Parsed: &RPCParsedInstruction{
									Type: "transferChecked",
									Info: map[string]any{
										"authority": "Owner11111111111111111111111111111111111111",
										"source":    "TokenAcct1111111111111111111111111111111111",
									},
								},
							}},
						},
					},
				}},
			},
		},
	}, ScrapeOptions{BlockLookback: 1, MaxWallets: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 || wallets[0] != "Owner11111111111111111111111111111111111111" {
		t.Fatalf("expected parsed transfer authority, got %#v", wallets)
	}
}

func TestScrapeRecentWalletsSkipsSPLTokenAccountSourceWithoutAuthority(t *testing.T) {
	wallets, err := ScrapeRecentWallets(context.Background(), fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {
				Transactions: []RPCBlockTxEnvelope{{
					Transaction: RPCTransaction{
						Message: RPCMessage{
							AccountKeys: []RPCAccountKey{{
								Pubkey:   "FeePayer1111111111111111111111111111111111",
								Signer:   true,
								Writable: true,
								Source:   "transaction",
							}},
							Instructions: []RPCInstruction{{
								Program: "spl-token",
								Parsed: &RPCParsedInstruction{
									Type: "transfer",
									Info: map[string]any{"source": "TokenAcct1111111111111111111111111111111111"},
								},
							}},
						},
					},
				}},
			},
		},
	}, ScrapeOptions{BlockLookback: 1, MaxWallets: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 0 {
		t.Fatalf("expected token-account source without authority to be skipped, got %#v", wallets)
	}
}

func TestScrapeRecentWalletsSkipsNativeDustBelowFloor(t *testing.T) {
	wallets, err := ScrapeRecentWallets(context.Background(), fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {
				Transactions: []RPCBlockTxEnvelope{{
					Transaction: RPCTransaction{
						Message: RPCMessage{
							Instructions: []RPCInstruction{{
								Program: "system",
								Parsed: &RPCParsedInstruction{
									Type: "transfer",
									Info: map[string]any{
										"lamports": float64(999),
										"source":   "DustSender111111111111111111111111111111111",
									},
								},
							}},
						},
					},
				}},
			},
		},
	}, ScrapeOptions{BlockLookback: 1, MaxWallets: 10, MinNativeLamports: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 0 {
		t.Fatalf("expected dust transfer to be skipped, got %#v", wallets)
	}
}

func TestQuietTransferLikeRejectsNoisyInstructions(t *testing.T) {
	instructions := []RPCInstruction{
		parsedIx("transfer"),
		parsedIx("swap"),
	}
	if quietTransferLike(instructions, 0) {
		t.Fatal("expected noisy transfer-like transaction to be rejected")
	}
	if !quietTransferLike(instructions, 1) {
		t.Fatal("expected noisy allowance to keep transaction")
	}
}

func TestScrapeRecentWalletsPrefersLowFrequencySigners(t *testing.T) {
	rpc := fakeRPC{
		slot: 10,
		blocks: map[int64]RPCBlock{
			10: {Transactions: []RPCBlockTxEnvelope{
				blockTx("Frequent11111111111111111111111111111111111", "transfer"),
				blockTx("Frequent11111111111111111111111111111111111", "transfer"),
				blockTx("Boring111111111111111111111111111111111111", "transfer"),
			}},
		},
	}

	wallets, err := ScrapeRecentWallets(context.Background(), rpc, ScrapeOptions{
		BlockLookback: 1,
		MaxWallets:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wallets[0] != "Boring111111111111111111111111111111111111" {
		t.Fatalf("expected low-frequency signer first, got %s", wallets[0])
	}
}

func blockTx(signer string, ixType string) RPCBlockTxEnvelope {
	return RPCBlockTxEnvelope{
		Transaction: RPCTransaction{
			Message: RPCMessage{
				AccountKeys: []RPCAccountKey{{
					Pubkey:   signer,
					Signer:   true,
					Writable: true,
					Source:   "transaction",
				}},
				Instructions: []RPCInstruction{{
					Program: "system",
					Parsed: &RPCParsedInstruction{
						Type: ixType,
						Info: map[string]any{
							"lamports": float64(100),
							"source":   signer,
						},
					},
				}},
			},
		},
	}
}

func parsedIx(ixType string) RPCInstruction {
	return RPCInstruction{Program: "system", Parsed: &RPCParsedInstruction{Type: ixType}}
}
