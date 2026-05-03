package response

import (
	"encoding/json"
	"testing"
)

func TestNewSetsAllFields(t *testing.T) {
	resp := New("healthy", "service is running", "proceed with requests")

	if resp.Status != "healthy" {
		t.Errorf("got status %q, want %q", resp.Status, Status("healthy"))
	}
	if resp.Reason != "service is running" {
		t.Errorf("got reason %q, want %q", resp.Reason, "service is running")
	}
	if resp.NextActionHint != "proceed with requests" {
		t.Errorf("got next_action_hint %q, want %q", resp.NextActionHint, "proceed with requests")
	}
}

func TestOKSetsStatusToOK(t *testing.T) {
	resp := OK("done", "continue")

	if resp.Status != StatusOK {
		t.Errorf("got status %q, want %q", resp.Status, StatusOK)
	}
}

func TestErrSetsStatusToError(t *testing.T) {
	resp := Err("failed", "retry later")

	if resp.Status != StatusError {
		t.Errorf("got status %q, want %q", resp.Status, StatusError)
	}
}

func TestResponseSerializesToJSON(t *testing.T) {
	resp := New(StatusOK, "all good", "send requests")

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if m["status"] != "ok" {
		t.Errorf("got status %q, want %q", m["status"], "ok")
	}
	if m["reason"] != "all good" {
		t.Errorf("got reason %q, want %q", m["reason"], "all good")
	}
	if m["next_action_hint"] != "send requests" {
		t.Errorf("got next_action_hint %q, want %q", m["next_action_hint"], "send requests")
	}
}

func TestResponseDeserializesFromJSON(t *testing.T) {
	input := `{"status":"ok","reason":"done","next_action_hint":"proceed"}`

	var resp Response
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Status != StatusOK {
		t.Errorf("got status %q, want %q", resp.Status, StatusOK)
	}
	if resp.Reason != "done" {
		t.Errorf("got reason %q, want %q", resp.Reason, "done")
	}
	if resp.NextActionHint != "proceed" {
		t.Errorf("got next_action_hint %q, want %q", resp.NextActionHint, "proceed")
	}
}
