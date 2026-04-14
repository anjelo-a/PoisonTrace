package wallets

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadAddressesFromFile(path string, max int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wallet file: %w", err)
	}
	defer f.Close()

	out := make([]string, 0, max)
	seen := make(map[string]struct{}, max)
	s := bufio.NewScanner(f)
	for s.Scan() {
		addr := strings.TrimSpace(s.Text())
		if addr == "" || strings.HasPrefix(addr, "#") {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
		if len(out) >= max {
			break
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read wallet file: %w", err)
	}
	return out, nil
}
