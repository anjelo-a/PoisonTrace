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
				Instructions: []RPCInstruction{parsedIx(ixType)},
			},
		},
	}
}

func parsedIx(ixType string) RPCInstruction {
	return RPCInstruction{Parsed: &RPCParsedInstruction{Type: ixType}}
}
