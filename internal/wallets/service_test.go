package wallets

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeAddressFile(t *testing.T, name string, addresses []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	data := ""
	for _, addr := range addresses {
		data += addr + "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write address file: %v", err)
	}
	return path
}

func TestLoadAddressesExcludingFileDedupesAndSkipsBeforeMax(t *testing.T) {
	walletFile := writeAddressFile(t, "wallets.txt", []string{
		"# comment",
		"walletA",
		"walletB",
		"walletA",
		"walletC",
		"walletD",
	})
	skipFile := writeAddressFile(t, "completed.txt", []string{
		"walletB",
	})

	got, err := LoadAddressesExcludingFile(walletFile, skipFile, 2)
	if err != nil {
		t.Fatalf("load addresses excluding file: %v", err)
	}
	want := []string{"walletA", "walletC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestLoadAddressesFromFileUnlimitedWhenMaxZero(t *testing.T) {
	walletFile := writeAddressFile(t, "wallets.txt", []string{"walletA", "walletB"})

	got, err := LoadAddressesFromFile(walletFile, 0)
	if err != nil {
		t.Fatalf("load addresses: %v", err)
	}
	want := []string{"walletA", "walletB"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
