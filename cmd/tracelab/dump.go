package main

import (
	"fmt"

	"tracelab/internal/normalize"
	"tracelab/internal/sink"
)

func runDump(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tracelab dump <traces.jsonl>")
	}
	records, err := sink.ReadAll(args[0])
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Println("(no records)")
		return nil
	}
	for i, rec := range records {
		printRecord(i, rec)
	}
	return nil
}

func printRecord(i int, rec sink.Record) {
	ev := rec.Event
	fmt.Printf("\n#%d  %s  kind=%s  session=%s\n", i, ev.ID, ev.Kind, ev.Correlation.SessionID)
	if rec.NormalizeError != "" {
		fmt.Printf("    %s\n", classifyUnmatched(rec))
		return
	}
	env := rec.Normalized
	if env == nil {
		return
	}
	fmt.Printf("    provider: %s  model: %s\n", env.Provider.Name, env.Provider.Model)
	if env.Request != nil {
		printSections(env.Request.Sections)
		fmt.Printf("    tools: %d  messages: %d\n", len(env.Request.Tools), len(env.Request.Messages))
	}
	if env.Response != nil && env.Response.Usage != nil {
		u := env.Response.Usage
		fmt.Printf("    usage: in=%d out=%d total=%d\n", u.InputTokens, u.OutputTokens, u.TotalTokens)
	}
	printSignals(env.Signals)
}

// classifyUnmatched explains why a record was not normalized, distinguishing
// benign control/probe traffic (a client pinging the base URL, listing models,
// etc.) from a genuine completion call that no provider recognized — the latter
// is worth attention, the former is noise.
func classifyUnmatched(rec sink.Record) string {
	ev := rec.Event
	if ev.Capture == nil {
		return "hook/non-http event (no LLM normalization)"
	}
	c := ev.Capture
	status := c.Response.StatusCode
	switch c.Method {
	case "GET", "HEAD", "OPTIONS":
		return fmt.Sprintf("control/probe (non-completion): %s %s -> %d", c.Method, c.Target, status)
	default:
		return fmt.Sprintf("UNMATCHED completion?: %s %s -> %d", c.Method, c.Target, status)
	}
}

func printSections(sections []normalize.SectionStat) {
	var total int64
	for _, s := range sections {
		total += s.ApproxTokens
	}
	if total == 0 {
		total = 1
	}
	fmt.Printf("    sections (approx tokens, %% of request):\n")
	for _, s := range sections {
		pct := float64(s.ApproxTokens) / float64(total) * 100
		fmt.Printf("      %-9s %8d  %5.1f%%\n", s.Name, s.ApproxTokens, pct)
	}
}

func printSignals(signals []normalize.TagSignal) {
	if len(signals) == 0 {
		return
	}
	fmt.Printf("    signals:")
	for _, s := range signals {
		tag := s.Tag
		if s.Suspected {
			tag = "~" + tag + "(疑似)"
		}
		fmt.Printf(" %s(%.0f%%)", tag, s.Confidence*100)
	}
	fmt.Println()
}
