package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// printResult writes a human-readable block for one scenario.
func printResult(r Result) {
	errPct := 0.0
	if r.Requests > 0 {
		errPct = float64(r.Errors) / float64(r.Requests) * 100
	}
	fmt.Printf("\n%-9s | %5.1fs | %s req | %s req/s | err %d (%.2f%%)\n",
		r.Scenario, r.WindowSec, comma(r.Requests), commaF(r.RPS), r.Errors, errPct)
	fmt.Printf("  latency  min %s  mean %s  p50 %s  p90 %s  p95 %s  p99 %s  p99.9 %s  max %s\n",
		ms(r.MinMs), ms(r.MeanMs), ms(r.P50Ms), ms(r.P90Ms), ms(r.P95Ms), ms(r.P99Ms), ms(r.P999Ms), ms(r.MaxMs))
	fmt.Printf("  status   %s\n", formatStatusCodes(r.StatusCodes))
	if r.Bytes > 0 {
		fmt.Printf("  rx       %.1f MB\n", float64(r.Bytes)/(1024*1024))
	}
	if r.Note != "" {
		fmt.Printf("  note     %s\n", r.Note)
	}
	if len(r.ErrSamples) > 0 {
		fmt.Printf("  errors   %s\n", strings.Join(capSamples(r.ErrSamples, 3), " | "))
	}
}

// printComparison prints a side-by-side table when several scenarios ran.
func printComparison(results []Result) {
	if len(results) < 2 {
		return
	}
	fmt.Printf("\n%s\n", strings.Repeat("=", 64))
	fmt.Printf("%-10s %14s %12s %12s %10s\n", "scenario", "req/s", "p50", "p99", "err%")
	fmt.Printf("%s\n", strings.Repeat("-", 64))
	for _, r := range results {
		errPct := 0.0
		if r.Requests > 0 {
			errPct = float64(r.Errors) / float64(r.Requests) * 100
		}
		fmt.Printf("%-10s %14s %12s %12s %9.2f%%\n",
			r.Scenario, commaF(r.RPS), ms(r.P50Ms), ms(r.P99Ms), errPct)
	}
	fmt.Printf("%s\n", strings.Repeat("=", 64))
}

func exportJSON(path string, results []Result) error {
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func exportCSV(path string, results []Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	defer cw.Flush()

	header := []string{"scenario", "concurrency", "window_sec", "requests", "success",
		"errors", "rps", "min_ms", "mean_ms", "p50_ms", "p90_ms", "p95_ms", "p99_ms", "p999_ms", "max_ms"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range results {
		row := []string{
			r.Scenario, strconv.Itoa(r.Concurrency), f6(r.WindowSec),
			strconv.FormatInt(r.Requests, 10), strconv.FormatInt(r.Success, 10),
			strconv.FormatInt(r.Errors, 10), f6(r.RPS),
			f6(r.MinMs), f6(r.MeanMs), f6(r.P50Ms), f6(r.P90Ms), f6(r.P95Ms), f6(r.P99Ms), f6(r.P999Ms), f6(r.MaxMs),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

func formatStatusCodes(codes map[int]int64) string {
	if len(codes) == 0 {
		return "(none)"
	}
	keys := make([]int, 0, len(codes))
	for k := range codes {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%d:%s", k, comma(codes[k])))
	}
	return strings.Join(parts, "  ")
}

func ms(v float64) string {
	switch {
	case v >= 1000:
		return fmt.Sprintf("%.2fs", v/1000)
	case v >= 10:
		return fmt.Sprintf("%.0fms", v)
	case v >= 1:
		return fmt.Sprintf("%.1fms", v)
	default:
		return fmt.Sprintf("%.2fms", v)
	}
}

func f6(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }

func capSamples(xs []string, n int) []string {
	if len(xs) <= n {
		return xs
	}
	return xs[:n]
}

// comma formats an integer with thousands separators.
func comma(v int64) string {
	s := strconv.FormatInt(v, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func commaF(v float64) string {
	return comma(int64(v+0.5)) + "." + fmt.Sprintf("%01d", int(v*10)%10)
}
