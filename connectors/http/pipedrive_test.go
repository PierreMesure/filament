package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/galaxy-io/filament"
	"github.com/galaxy-io/filament/checkpoint"
)

func pipedriveTestSource(t *testing.T, handler http.HandlerFunc) *Source {
	t.Helper()
	api := httptest.NewServer(handler)
	t.Cleanup(api.Close)
	data := strings.Replace(string(pipedriveManifest), "requests_per_second: 5", "requests_per_second: 1000", 1)
	src := NewManifest([]byte(data))
	if err := src.Configure(t.Context(), filament.NewConfig(map[string]any{
		"api_token": "tok_test",
		"host":      api.URL,
	})); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Teardown(t.Context()) })
	return src
}

// pipedriveRow decodes an emitted row without losing integer precision.
func pipedriveRow(t *testing.T, data []byte) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var row map[string]any
	if err := dec.Decode(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestNewPipedriveSpecAndEmbeddedManifest(t *testing.T) {
	ctx := t.Context()
	src := NewPipedrive()
	spec := src.Spec()
	if spec.Name != "pipedrive" || spec.DisplayName != "Pipedrive" {
		t.Fatalf("spec identity = %q/%q", spec.Name, spec.DisplayName)
	}
	if spec.DarkLogoURL != "https://cdn.getgalaxy.io/sources/source-icon-pipedrive-dark.svg" ||
		spec.LightLogoURL != "https://cdn.getgalaxy.io/sources/source-icon-pipedrive-light.svg" {
		t.Fatalf("logos = %q / %q", spec.DarkLogoURL, spec.LightLogoURL)
	}
	if len(spec.Config.Fields) != 2 {
		t.Fatalf("config fields = %#v, want api_token and host", spec.Config.Fields)
	}
	token, host := spec.Config.Fields[0], spec.Config.Fields[1]
	if token.Name != "api_token" || token.Type != filament.FieldSecret || !token.Required {
		t.Fatalf("api_token field = %#v, want required secret", token)
	}
	if host.Name != "host" || host.Type != filament.FieldString || host.Required {
		t.Fatalf("host field = %#v, want optional string", host)
	}
	if err := src.Validate(filament.NewConfig(map[string]any{"host": "https://example.pipedrive.com"})); err == nil {
		t.Fatal("validate without API token succeeded")
	}
	// The host default must make the token the only required input.
	if err := src.Configure(ctx, filament.NewConfig(map[string]any{"api_token": "tok_test"})); err != nil {
		t.Fatal(err)
	}
	defer src.Teardown(ctx)

	discovered, err := src.Discover(ctx, filament.DiscoverOpts{})
	if err != nil {
		t.Fatal(err)
	}
	var names, enabled []string
	for _, resource := range discovered.Resources {
		names = append(names, resource.Name)
		if resource.Metadata["default_resources"] == "true" {
			enabled = append(enabled, resource.Name)
		}
	}
	wantNames := []string{
		"users", "deals", "deals_archived", "persons", "organizations", "activities",
		"leads", "leads_archived", "notes", "products", "pipelines", "stages",
		"deal_fields", "person_fields", "organization_fields", "activity_fields", "product_fields",
		"activity_types", "lead_labels", "currencies", "filters", "projects", "tasks", "files",
	}
	wantEnabled := []string{
		"users", "deals", "persons", "organizations", "activities", "leads", "notes",
		"pipelines", "stages", "deal_fields", "person_fields", "organization_fields",
		"activity_types", "lead_labels",
	}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("resources = %v", names)
	}
	if !slices.Equal(enabled, wantEnabled) {
		t.Fatalf("defaults = %v", enabled)
	}
	for name, key := range map[string]string{"deals": "id", "leads": "id", "deal_fields": "field_code", "lead_labels": "id"} {
		schema, err := src.Schema(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(schema.PrimaryKey, []string{key}) {
			t.Fatalf("%s primary key = %v, want %s", name, schema.PrimaryKey, key)
		}
	}
	incremental := []string{"deals", "deals_archived", "persons", "organizations", "activities", "products", "leads"}
	for _, name := range names {
		cols, err := src.CursorColumns(ctx, name)
		if slices.Contains(incremental, name) {
			if err != nil || len(cols) != 1 || cols[0].Name != "update_time" {
				t.Fatalf("%s cursor columns = %#v, %v", name, cols, err)
			}
			continue
		}
		if err != nil || len(cols) != 0 {
			t.Fatalf("%s advertised incremental cursor %#v, %v", name, cols, err)
		}
	}
	// Notes are full-only: Pipedrive v1 note timestamps are not RFC3339, so
	// the time comparator cannot order them.
	if _, err := src.PlanIncremental(ctx, []string{"notes"}, nil, nil); err == nil {
		t.Fatal("incremental plan for notes succeeded")
	}
}

func TestPipedriveExtractsCursorOffsetAndUnpaginatedLists(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	const deal = `{"id":%s,"title":"Big contract","status":"open","value":2300.5,"currency":"EUR","owner_id":8,"person_id":12,"org_id":3,"pipeline_id":1,"stage_id":2,"probability":null,"expected_close_date":"2026-04-30","add_time":"2026-03-01T10:00:00Z","update_time":"2026-03-01T12:00:00Z","stage_change_time":null,"won_time":null,"lost_time":null,"close_time":null,"lost_reason":null,"visible_to":3,"is_archived":false,"is_deleted":false,"label_ids":[1,2],"origin":"ManuallyCreated","origin_id":null,"source_lead_id":null,"acv":null,"custom_fields":{"dcf558aac1ae4e8c4f849ba5e668430d8df9be12":{"value":2300,"currency":"EUR"}}}`
	src := pipedriveTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		q := r.URL.Query()
		if r.Method != http.MethodGet || r.Header.Get("x-api-token") != "tok_test" || r.Header.Get("Accept") != "application/json" || q.Has("api_token") {
			t.Errorf("unexpected request: %s %s headers=%v", r.Method, r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/users":
			if len(q) != 0 {
				t.Errorf("users query = %v, want none", q)
			}
			fmt.Fprint(w, `{"success":true,"data":[{"id":8,"name":"Ada","email":"ada@example.com","active_flag":true,"is_deleted":false,"role_id":1,"created":"2019-01-22 08:55:59","modified":"2026-02-01 09:00:00","last_login":"2026-03-01 07:30:00","access":[{"app":"sales","admin":true}]},{"id":9,"name":"Grace","email":"grace@example.com","active_flag":false,"created":"2019-01-22 08:55:59","modified":null}]}`)
		case "/api/v2/deals":
			if q.Get("limit") != "500" || q.Has("updated_since") {
				t.Errorf("full deal query = %v", q)
			}
			switch q.Get("cursor") {
			case "":
				fmt.Fprintf(w, `{"success":true,"data":[%s],"additional_data":{"next_cursor":"eyJkZWFsIjoxfQ"}}`, fmt.Sprintf(deal, "9007199254740993"))
			case "eyJkZWFsIjoxfQ":
				fmt.Fprintf(w, `{"success":true,"data":[%s],"additional_data":{"next_cursor":null}}`, fmt.Sprintf(deal, "2"))
			default:
				t.Errorf("deal cursor = %q", q.Get("cursor"))
			}
		case "/api/v1/leads":
			if q.Get("limit") != "100" || q.Has("updated_since") {
				t.Errorf("lead query = %v", q)
			}
			var leads []string
			switch q.Get("start") {
			case "0":
				for i := 0; i < 100; i++ {
					leads = append(leads, fmt.Sprintf(`{"id":"adf21080-0e10-11eb-9a5f-%012d","title":"Lead %d","owner_id":8,"creator_id":8,"person_id":12,"organization_id":null,"label_ids":[],"value":{"amount":100,"currency":"USD"},"expected_close_date":null,"is_archived":false,"was_seen":true,"add_time":"2026-02-01T09:00:00.000Z","update_time":"2026-02-02T09:00:00.551Z","dcf558aac1ae4e8c4f849ba5e668430d8df9be12":"custom"}`, i, i))
				}
			case "100":
				leads = append(leads, `{"id":"adf21080-0e10-11eb-9a5f-ffffffffffff","title":"Last lead","owner_id":8,"creator_id":8,"add_time":"2026-02-01T09:00:00.000Z","update_time":"2026-02-02T09:00:00.551Z"}`)
			default:
				t.Errorf("lead start = %q, want the walk to stop after a short page", q.Get("start"))
			}
			fmt.Fprintf(w, `{"success":true,"data":[%s],"additional_data":{"pagination":{"start":%s,"limit":100,"more_items_in_collection":%t}}}`, strings.Join(leads, ","), q.Get("start"), len(leads) == 100)
		case "/api/v1/notes":
			if q.Get("start") != "0" || q.Get("limit") != "100" {
				t.Errorf("note query = %v", q)
			}
			fmt.Fprint(w, `{"success":true,"data":[{"id":55,"content":"<p>Called back</p>","user_id":8,"deal_id":2,"person_id":null,"org_id":null,"lead_id":null,"active_flag":true,"pinned_to_deal_flag":true,"add_time":"2026-03-01 11:00:00","update_time":"2026-03-01 11:05:00","deal":{"title":"Big contract"},"user":{"email":"ada@example.com","name":"Ada"}}],"additional_data":{"pagination":{"start":0,"limit":100,"more_items_in_collection":false}}}`)
		case "/api/v2/dealFields":
			if q.Get("limit") != "500" {
				t.Errorf("deal field query = %v", q)
			}
			fmt.Fprint(w, `{"success":true,"data":[{"field_code":"dcf558aac1ae4e8c4f849ba5e668430d8df9be12","field_name":"Contract value","field_type":"monetary","is_custom_field":true,"is_optional_response_field":false,"options":null,"subfields":null,"ui_visibility":{"add":true}}],"additional_data":{"next_cursor":null}}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})

	selected := []string{"users", "deals", "leads", "notes", "deal_fields"}
	var sink collectSink
	if err := src.Extract(t.Context(), &sink, filament.ExtractOpts{Resources: selected}); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, rec := range sink.records {
		counts[rec.Resource]++
		if !slices.Contains(selected, rec.Resource) {
			t.Fatalf("unselected resource emitted: %s", rec.Resource)
		}
		row := pipedriveRow(t, rec.Data)
		raw, _ := row["raw"].(map[string]any)
		switch rec.Resource {
		case "users":
			if row["created"] == nil || (row["name"] == "Ada" && raw["access"] == nil) {
				t.Fatalf("user projection = %#v", row)
			}
		case "deals":
			if id := row["id"].(json.Number).String(); id != "9007199254740993" && id != "2" {
				t.Fatalf("deal id = %s, want exact int64", id)
			}
			custom, _ := row["custom_fields"].(map[string]any)
			if custom["dcf558aac1ae4e8c4f849ba5e668430d8df9be12"] == nil || row["probability"] != nil || row["value"].(json.Number).String() != "2300.5" {
				t.Fatalf("deal projection = %#v", row)
			}
			if _, kept := raw["acv"]; !kept || raw["title"] != nil {
				t.Fatalf("deal raw remainder = %#v", raw)
			}
		case "leads":
			if row["title"] == "Lead 0" && (raw["dcf558aac1ae4e8c4f849ba5e668430d8df9be12"] != "custom" || row["value"].(map[string]any)["amount"] == nil) {
				t.Fatalf("lead projection = %#v", row)
			}
		case "notes":
			if row["update_time"] == nil || raw["deal"].(map[string]any)["title"] != "Big contract" {
				t.Fatalf("note projection = %#v", row)
			}
		case "deal_fields":
			if row["field_code"] != "dcf558aac1ae4e8c4f849ba5e668430d8df9be12" || row["options"] != nil || raw["ui_visibility"] == nil {
				t.Fatalf("deal field projection = %#v", row)
			}
		}
	}
	for name, want := range map[string]int{"users": 2, "deals": 2, "leads": 101, "notes": 1, "deal_fields": 1} {
		if counts[name] != want {
			t.Fatalf("%s rows = %d, want %d", name, counts[name], want)
		}
	}
	mu.Lock()
	if hits["/api/v2/deals"] != 2 || hits["/api/v1/leads"] != 2 || hits["/api/v1/users"] != 1 {
		t.Fatalf("request counts = %v", hits)
	}
	hits = map[string]int{}
	mu.Unlock()

	// A bulk resource on its own must not drag any other endpoint along.
	var only collectSink
	if err := src.Extract(t.Context(), &only, filament.ExtractOpts{Resources: []string{"deals"}}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(hits) != 1 || hits["/api/v2/deals"] != 2 || len(only.records) != 2 {
		t.Fatalf("deals-only requests = %v rows = %d", hits, len(only.records))
	}
}

func TestPipedriveIncrementalInjectsUpdatedSinceAndAdvancesWatermark(t *testing.T) {
	ctx := t.Context()
	src := pipedriveTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// The stored watermark is rewound by the 60 second overlap and must stay
		// fixed across every page of the walk.
		if r.URL.Path != "/api/v2/deals" || q.Get("updated_since") != "2026-03-01T11:59:00Z" || q.Get("limit") != "500" {
			t.Errorf("incremental request = %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if q.Get("cursor") == "" {
			fmt.Fprint(w, `{"success":true,"data":[{"id":1,"title":"Renewal","add_time":"2026-03-01T10:00:00Z","update_time":"2026-03-01T12:30:00Z","custom_fields":{}}],"additional_data":{"next_cursor":"bmV4dA"}}`)
			return
		}
		if q.Get("cursor") != "bmV4dA" {
			t.Errorf("cursor = %q", q.Get("cursor"))
		}
		fmt.Fprint(w, `{"success":true,"data":[{"id":2,"title":"Expansion","add_time":"2026-03-01T10:00:00Z","update_time":"2026-03-01T12:10:00Z","custom_fields":{}}],"additional_data":{"next_cursor":null}}`)
	})
	prev := map[string]filament.Checkpoint{
		"deals": checkpoint.KeysetCheckpoint{
			Mode: checkpoint.ModeIncremental, Cols: []string{"deals_update_time"}, Types: []string{"timestamptz"},
			Shards: []checkpoint.KeysetShard{{Key: []string{"2026-03-01T12:00:00Z"}}},
		}.ToCheckpoint("deals"),
	}
	plan, err := src.PlanIncremental(ctx, []string{"deals"}, prev, nil)
	if err != nil {
		t.Fatal(err)
	}
	var sink collectSink
	if err := src.ExtractFrom(ctx, &sink, filament.ExtractOpts{Resources: []string{"deals"}}, plan); err != nil {
		t.Fatal(err)
	}
	if len(sink.records) != 2 {
		t.Fatalf("rows = %d, want 2", len(sink.records))
	}
	for _, rec := range sink.records {
		if !slices.Equal(rec.Key, []string{"2026-03-01T12:30:00Z"}) {
			t.Fatalf("watermark = %v, want the maximum update_time without a page cursor", rec.Key)
		}
	}
}

func TestPipedriveIncrementalFailsOnMissingUpdateTime(t *testing.T) {
	// A record without update_time must fail the run rather than be skipped:
	// silently dropping it would advance the watermark past an unread change.
	src := pipedriveTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":[{"id":1,"title":"No timestamp","add_time":"2026-03-01T10:00:00Z","update_time":null}],"additional_data":{"next_cursor":null}}`)
	})
	plan, err := src.PlanIncremental(t.Context(), []string{"persons"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var sink collectSink
	if err := src.ExtractFrom(t.Context(), &sink, filament.ExtractOpts{Resources: []string{"persons"}}, plan); err == nil {
		t.Fatalf("null update_time was accepted, rows=%d", len(sink.records))
	}
}

func TestPipedriveTestConnectionProbesUsers(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	unauthorized := false
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if r.Header.Get("x-api-token") != "tok_test" {
			t.Errorf("probe headers = %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		if unauthorized {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"success":false,"error":"unauthorized access","errorCode":401}`)
			return
		}
		fmt.Fprint(w, `{"success":true,"data":[]}`)
	}))
	defer api.Close()
	src := NewPipedrive()
	cfg := filament.NewConfig(map[string]any{"api_token": "tok_test", "host": api.URL})
	if err := src.TestConnection(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if !slices.Equal(paths, []string{"/api/v1/users"}) {
		t.Fatalf("probe paths = %v, want one users request", paths)
	}
	unauthorized = true
	mu.Unlock()
	if err := src.TestConnection(t.Context(), cfg); err == nil {
		t.Fatal("rejected token passed the connection probe")
	}
}

func TestPipedriveEmptyV1ListWithNullDataYieldsNoRows(t *testing.T) {
	// Observed live: v1 notes and files answer {"data": null} when the account
	// has none, while a non-array of any other kind is still an error.
	for _, tc := range []struct {
		name, body string
		wantErr    bool
	}{
		{"null data", `{"success":true,"data":null,"additional_data":{"pagination":{"start":0,"limit":100,"more_items_in_collection":false}}}`, false},
		{"object data", `{"success":true,"data":{"id":1}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := pipedriveTestSource(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.body)
			})
			var sink collectSink
			err := src.Extract(t.Context(), &sink, filament.ExtractOpts{Resources: []string{"notes"}})
			if (err != nil) != tc.wantErr || len(sink.records) != 0 {
				t.Fatalf("rows=%d err=%v", len(sink.records), err)
			}
		})
	}
}
