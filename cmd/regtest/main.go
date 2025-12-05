package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/store"
)

func seedNotes(s *store.Store) ([]string, map[string]string, error) {

	tokens := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	now := time.Now().UTC()
	ids := make([]string, 0, 20)
	tokenForID := make(map[string]string, 20)

	for i := 1; i <= 20; i++ {
		id := fmt.Sprintf("n%04d", i)
		tok := tokens[(i-1)%len(tokens)]
		title := fmt.Sprintf("Note %d", i)
		text := fmt.Sprintf("This is test note %d. token=%s. boilerplate text.", i, tok)
		n := &model.Note{
			ID:        id,
			Title:     title,
			Text:      text,
			Tags:      []string{"reg"},
			CreatedAt: now,
			UpdatedAt: now,
			Deleted:   false,
		}
		if err := s.Append(n); err != nil {
			return nil, nil, fmt.Errorf("append %s: %w", id, err)
		}
		ids = append(ids, id)
		tokenForID[id] = tok
	}
	return ids, tokenForID, nil
}

func containsAll(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}

func equalSets(a, b []string) bool {
	return containsAll(a, b) && containsAll(b, a)
}

func main() {
	_ = i18n.InitFromEnv()

	home, _ := os.UserHomeDir()
	tmpRoot, err := os.MkdirTemp(home, "data_regtest-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("regtest.err_tempdir", err))
		os.Exit(2)
	}

	defer func() {
		_ = os.RemoveAll(tmpRoot)
	}()

	fmt.Printf(i18n.T("regtest.using_store")+"\n", tmpRoot)

	s, err := store.Open(tmpRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("regtest.err_open_store", err))
		os.Exit(3)
	}
	defer s.Close()

	ids, tokenForID, err := seedNotes(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("regtest.err_seed", err))
		os.Exit(4)
	}

	fmt.Printf(i18n.T("regtest.seeded")+"\n", len(ids))

	fmt.Println(i18n.T("regtest.exact_checks_title"))

	total := 0
	passed := 0
	for _, id := range ids {
		total++
		n, err := s.GetByID(id)
		if err != nil {
			fmt.Printf(i18n.T("regtest.fail_exact_not_found")+"\n", id, err)
			continue
		}
		if n.ID != id || n.Deleted {
			fmt.Printf(i18n.T("regtest.fail_exact_unexpected")+"\n", id, n.ID, n.Deleted)
			continue
		}
		passed++

		fmt.Printf(i18n.T("regtest.pass_exact")+"\n", id)

	}

	fmt.Println(i18n.T("regtest.contains_checks_title"))

	tokenBuckets := map[string][]string{}
	for id, tok := range tokenForID {
		tokenBuckets[tok] = append(tokenBuckets[tok], id)
	}
	tokens := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for _, tok := range tokens {
		total++
		exp := tokenBuckets[tok]

		list, err := s.List(store.Filter{Contains: tok, Limit: 100})
		if err != nil {
			fmt.Printf(i18n.T("regtest.fail_contains_query_error")+"\n", tok, err)
			continue
		}
		got := make([]string, 0, len(list))
		for _, n := range list {
			got = append(got, n.ID)
		}

		if equalSets(exp, got) {
			passed++
			fmt.Printf(i18n.T("regtest.pass_contains")+"\n", tok, len(got))
		} else {
			fmt.Printf(i18n.T("regtest.fail_contains")+"\n", tok)
			fmt.Printf(i18n.T("regtest.expect")+"\n", exp)
			fmt.Printf(i18n.T("regtest.got")+"\n", got)
		}
	}

	fmt.Printf(i18n.T("regtest.summary")+"\n", passed, total)
	if passed != total {
		fmt.Printf(i18n.T("regtest.failed") + "\n")
		os.Exit(5)
	}
	fmt.Printf(i18n.T("regtest.all_passed")+"\n", filepath.Base(tmpRoot))
}
