package httpapi

import (
	"encoding/json"
	"os"
	"slices"
	"sort"
	"testing"

	"github.com/galaxy-io/filament"
	"github.com/galaxy-io/filament/checkpoint"
)

// TestPipedriveLive runs a bounded read-only smoke test against a real account.
// Set FILAMENT_TEST_PIPEDRIVE_TOKEN and optionally FILAMENT_TEST_PIPEDRIVE_HOST.
func TestPipedriveLive(t *testing.T) {
	token := os.Getenv("FILAMENT_TEST_PIPEDRIVE_TOKEN")
	if token == "" {
		t.Skip("FILAMENT_TEST_PIPEDRIVE_TOKEN not set")
	}
	cfg := map[string]any{"api_token": token}
	if host := os.Getenv("FILAMENT_TEST_PIPEDRIVE_HOST"); host != "" {
		cfg["host"] = host
	}
	ctx := t.Context()
	src := NewPipedrive()
	if err := src.TestConnection(ctx, filament.NewConfig(cfg)); err != nil {
		t.Fatalf("connection probe: %v", err)
	}
	if err := src.TestConnection(ctx, filament.NewConfig(map[string]any{"api_token": "invalid", "host": cfg["host"]})); err == nil {
		t.Fatal("invalid token passed the connection probe")
	}
	if err := src.Configure(ctx, filament.NewConfig(cfg)); err != nil {
		t.Fatal(err)
	}
	defer src.Teardown(ctx)
	discovered, err := src.Discover(ctx, filament.DiscoverOpts{})
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for _, r := range discovered.Resources {
		all = append(all, r.Name)
	}

	var sink collectSink
	if err := src.Extract(ctx, &sink, filament.ExtractOpts{Resources: all}); err != nil {
		t.Fatalf("full extract: %v", err)
	}
	counts := map[string]int{}
	watermarks := map[string]string{}
	for _, rec := range sink.records {
		counts[rec.Resource]++
		var row map[string]any
		if err := json.Unmarshal(rec.Data, &row); err != nil {
			t.Fatal(err)
		}
		if row["raw"] == nil {
			t.Fatalf("%s row without raw remainder: %s", rec.Resource, rec.Data)
		}
		if ut, ok := row["update_time"].(string); ok && ut > watermarks[rec.Resource] {
			watermarks[rec.Resource] = ut
		}
		if counts[rec.Resource] == 1 {
			t.Logf("%s sample: %s", rec.Resource, truncate(string(rec.Data), 400))
		}
	}
	names := slices.Clone(all)
	sort.Strings(names)
	for _, name := range names {
		t.Logf("%-22s %d rows", name, counts[name])
	}

	// Incremental replay from each resource's observed maximum update_time
	// must return the boundary rows again (inclusive filter plus overlap) and
	// nothing older.
	for _, name := range []string{"deals", "persons", "organizations", "activities", "leads", "products"} {
		mark := watermarks[name]
		if mark == "" {
			t.Logf("%s: no rows, skipping incremental replay", name)
			continue
		}
		prev := map[string]filament.Checkpoint{
			name: checkpoint.KeysetCheckpoint{
				Mode: checkpoint.ModeIncremental, Cols: []string{name + "_update_time"}, Types: []string{"timestamptz"},
				Shards: []checkpoint.KeysetShard{{Key: []string{mark}}},
			}.ToCheckpoint(name),
		}
		plan, err := src.PlanIncremental(ctx, []string{name}, prev, nil)
		if err != nil {
			t.Fatalf("%s plan incremental: %v", name, err)
		}
		var incr collectSink
		if err := src.ExtractFrom(ctx, &incr, filament.ExtractOpts{Resources: []string{name}}, plan); err != nil {
			t.Fatalf("%s incremental extract: %v", name, err)
		}
		if len(incr.records) == 0 || len(incr.records) > counts[name] {
			t.Fatalf("%s incremental rows = %d (full = %d)", name, len(incr.records), counts[name])
		}
		for _, rec := range incr.records {
			if !slices.Equal(rec.Key, []string{mark}) && (len(rec.Key) != 1 || rec.Key[0] < mark) {
				t.Fatalf("%s incremental watermark regressed: %v < %s", name, rec.Key, mark)
			}
		}
		t.Logf("%-14s incremental from %s: %d rows (full %d)", name, mark, len(incr.records), counts[name])
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
