package main

// Simple benchmark runner for NoteLine:
// - can generate N notes (fast) into the store (Append -> also indexes)
// - run Q random or supplied queries and measure latencies
// - prints summary (total indexing time, avg search latency, p50/p95)
// This version runs the whole benchmark inside an isolated temporary copy of the store root,
// and removes that temporary directory at the end — original data is not modified.

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/store"
)

func run() int {
	_ = i18n.InitFromEnv()
	var n int
	var q int
	var outCSV string

	// flag.StringVar(&root, "root", "./data_tmp", "store root")
	flag.IntVar(&n, "n", 1000, "number of notes to create")
	flag.IntVar(&q, "q", 100, "number of search queries to run")
	flag.StringVar(&outCSV, "out", "", "optional CSV file to write per-query latencies")
	flag.Parse()

	// fmt.Printf("bench: requested root=%s n=%d q=%d seed=%d\n", root, n, q, seed)

	// Create temporary isolated root.
	home, _ := os.UserHomeDir()
	tmpRoot, err := os.MkdirTemp(home, "noteline-bench-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("bench.err_tempdir", err))
		return 2
	}
	// Ensure removal of temp dir when run() returns (will run before os.Exit in main).
	defer func() {
		_ = os.RemoveAll(tmpRoot)
	}()

	// Use tmpRoot as working root for this benchmark run
	workRoot := tmpRoot
	fmt.Printf(i18n.T("bench.running_root")+"\n", workRoot)

	// open store on isolated root
	s, err := store.Open(workRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("bench.err_open_store", err))
		return 3
	}
	// ensure store closed before leaving run() so files are flushed.
	defer func() {
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", i18n.T("bench.warning_store_close", err))
		}
	}()

	// 1) Generate N notes
	fmt.Printf(i18n.T("bench.creating_notes")+"\n", n)

	start := time.Now()
	// ids := make([]string, 0, n)
	for i := 0; i < n; i++ { ///////////////////////////////////
		title := fmt.Sprintf("Bench note %d", i)
		token := fmt.Sprintf("%s_%04d", "base_text", i%100) // reuse tokens to test hit lists
		text := fmt.Sprintf("This is benchmark note %d containing token %s and some junk text", i, token)
		note := model.NewNote(title, text, []string{"bench", strconv.Itoa(i % 10)})
		if err := s.Append(note); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", i18n.T("bench.append_note_error", i, err))
			return 4
		}
		// ids = append(ids, note.ID)
	}
	indexDur := time.Since(start)
	if n > 0 {
		fmt.Printf(i18n.T("bench.indexing_done_avg")+"\n", indexDur, indexDur/time.Duration(n))
	} else {
		fmt.Printf(i18n.T("bench.indexing_done")+"\n", indexDur)
	}

	// 2) Warm-up few queries
	fmt.Printf(i18n.T("bench.warmup") + "\n")
	for i := 0; i < 10 && i < q; i++ {
		tok := fmt.Sprintf("%s_%04d", "base_text", rand.Intn(100))
		_, _ = s.List(store.Filter{Contains: tok, Limit: 10})
	}

	// 3) Run Q queries and measure latencies
	latencies := make([]time.Duration, 0, q)
	var hitsTotal int
	for i := 0; i < q; i++ {
		tok := fmt.Sprintf("%s_%04d", "base_text", rand.Intn(100))
		t0 := time.Now()
		list, _ := s.List(store.Filter{Contains: tok, Limit: 100})
		d := time.Since(t0)
		latencies = append(latencies, d)
		hitsTotal += len(list)
		if i%50 == 0 {
			fmt.Printf(i18n.T("bench.query_sample")+"\n", i, q, tok, d, len(list))
		}

	}

	// sort latencies for percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	sum := time.Duration(0)
	for _, v := range latencies {
		sum += v
	}
	avg := time.Duration(0)
	if len(latencies) > 0 {
		avg = sum / time.Duration(len(latencies))
	}

	p50 := time.Duration(0)
	p95 := time.Duration(0)
	p99 := time.Duration(0)
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)/2]
		p95 = latencies[(len(latencies)*95)/100]
		p99 = latencies[(len(latencies)*99)/100]
	}

	fmt.Println(i18n.T("bench.summary_title"))
	fmt.Printf(i18n.T("bench.notes_created")+"\n", n)
	fmt.Printf(i18n.T("bench.index_duration")+"\n", indexDur)
	fmt.Printf(i18n.T("bench.queries_run")+"\n", q)
	fmt.Printf(i18n.T("bench.avg_latency")+"\n", avg)
	fmt.Printf(i18n.T("bench.percentiles")+"\n", p50, p95, p99)
	if q > 0 {
		// bench.avg_hits expects one string argument (formatted)
		avgHitsStr := fmt.Sprintf("%.2f", float64(hitsTotal)/float64(q))
		fmt.Printf(i18n.T("bench.avg_hits")+"\n", avgHitsStr)
	} else {
		fmt.Printf(i18n.T("bench.avg_hits")+"\n", fmt.Sprintf("%d", hitsTotal))
	}

	// optional CSV output
	if outCSV != "" {
		f, err := os.Create(outCSV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", i18n.T("bench.err_create_csv", err))
			return 5
		}
		w := csv.NewWriter(f)
		_ = w.Write([]string{"query_index", "latency_ns"})
		for i, v := range latencies {
			_ = w.Write([]string{strconv.Itoa(i), strconv.FormatInt(int64(v), 10)})
		}
		w.Flush()
		_ = f.Close()
		fmt.Printf(i18n.T("bench.wrote_csv")+"\n", outCSV)
	}

	// small disk-usage note (for the isolated copy)
	idxPath := filepath.Join(workRoot, "index.bleve")
	if fi, err := os.Stat(idxPath); err == nil {
		fmt.Printf(i18n.T("bench.index_path_size")+"\n", idxPath, fi.Size())
	} else {
		fmt.Printf(i18n.T("bench.index_path_stat_failed")+"\n", idxPath, err)
	}

	// No need to mark notes deleted: we used an isolated copy and will remove it.
	fmt.Printf(i18n.T("bench.finished_removing")+"\n", workRoot)
	// deferred RemoveAll will remove it when run() returns.

	return 0
}

func main() {
	os.Exit(run())
}
