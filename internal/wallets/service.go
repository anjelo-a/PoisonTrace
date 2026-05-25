package wallets

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadAddressesFromFile(path string, max int) ([]string, error) {
	return LoadAddressesExcludingFile(path, "", max)
}

func LoadAddressesExcludingFile(path string, excludePath string, max int) ([]string, error) {
	excluded := map[string]struct{}{}
	if excludePath != "" {
		exclude, err := LoadAddressesFromFile(excludePath, 0)
		if err != nil {
			return nil, fmt.Errorf("load excluded wallet file: %w", err)
		}
		for _, addr := range exclude {
			excluded[addr] = struct{}{}
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wallet file: %w", err)
	}
	defer f.Close()

	out := make([]string, 0)
	seen := make(map[string]struct{})
	s := bufio.NewScanner(f)
	for s.Scan() {
		addr := strings.TrimSpace(s.Text())
		if addr == "" || strings.HasPrefix(addr, "#") {
			continue
		}
		if _, ok := excluded[addr]; ok {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
		if max > 0 && len(out) >= max {
			break
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read wallet file: %w", err)
	}
	return out, nil
}

func NormalizeAddresses(addresses []string, max int) ([]string, int) {
	out := make([]string, 0, len(addresses))
	seen := make(map[string]struct{})
	for _, raw := range addresses {
		addr := strings.TrimSpace(raw)
		if addr == "" || strings.HasPrefix(addr, "#") {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		if max > 0 && len(out) >= max {
			continue
		}
		out = append(out, addr)
	}
	return out, len(seen)
}
