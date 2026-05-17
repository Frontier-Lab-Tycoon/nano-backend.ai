package response

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNewSetsAllFields(t *testing.T) {
	errPayload := &Error{Code: "validation_error", Message: "missing field"}
	resp := New(http.StatusUnprocessableEntity, nil, errPayload)

	if resp.Status != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d", resp.Status, http.StatusUnprocessableEntity)
	}
	if resp.Data != nil {
		t.Errorf("got data %v, want nil", resp.Data)
	}
	if resp.Error != errPayload {
		t.Errorf("got error %v, want original error payload", resp.Error)
	}
}

func TestOKSetsStatusToOK(t *testing.T) {
	resp := OK(map[string]string{"state": "healthy"})

	if resp.Status != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Error != nil {
		t.Errorf("got error %v, want nil", resp.Error)
	}
}

func TestSuccessUsesGivenStatusCode(t *testing.T) {
	resp := Success(http.StatusCreated, map[string]string{"id": "run_123"})

	if resp.Status != http.StatusCreated {
		t.Errorf("got status %d, want %d", resp.Status, http.StatusCreated)
	}
}

func TestOKUsesEmptyObjectWhenDataIsNil(t *testing.T) {
	resp := OK(nil)

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if _, ok := m["data"].(map[string]any); !ok {
		t.Fatalf("got data %T, want object", m["data"])
	}
}

func TestErrSetsStatusToError(t *testing.T) {
	resp := Err(http.StatusNotFound, "not_found", "run not found", "check the run ID and retry", nil)

	if resp.Status != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.Status, http.StatusNotFound)
	}
	if resp.Error == nil {
		t.Fatal("got nil error payload, want error payload")
	}
	if resp.Error.Code != "not_found" {
		t.Errorf("got code %q, want %q", resp.Error.Code, "not_found")
	}
}

func TestResponseSerializesToJSON(t *testing.T) {
	resp := OK(map[string]string{"state": "healthy"})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if m["status"] != float64(http.StatusOK) {
		t.Errorf("got status %v, want %d", m["status"], http.StatusOK)
	}
	dataPayload, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatalf("got data %T, want object", m["data"])
	}
	if dataPayload["state"] != "healthy" {
		t.Errorf("got state %q, want %q", dataPayload["state"], "healthy")
	}
}

func TestErrorResponseSerializesToJSON(t *testing.T) {
	resp := Err(http.StatusUnprocessableEntity, "validation_error", "missing required field", "provide project_id and retry", map[string]string{
		"field": "project_id",
	})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if m["status"] != float64(http.StatusUnprocessableEntity) {
		t.Errorf("got status %v, want %d", m["status"], http.StatusUnprocessableEntity)
	}
	errorPayload, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("got error %T, want object", m["error"])
	}
	if errorPayload["code"] != "validation_error" {
		t.Errorf("got code %q, want %q", errorPayload["code"], "validation_error")
	}
	details, ok := errorPayload["details"].(map[string]any)
	if !ok {
		t.Fatalf("got details %T, want object", errorPayload["details"])
	}
	if details["field"] != "project_id" {
		t.Errorf("got field %q, want %q", details["field"], "project_id")
	}
}

func TestResponseDeserializesFromJSON(t *testing.T) {
	input := `{"status":404,"error":{"code":"not_found","message":"run not found","next_action_hint":"check the run ID and retry"}}`

	var resp Response
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Status != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.Status, http.StatusNotFound)
	}
	if resp.Error == nil {
		t.Fatal("got nil error payload, want error payload")
	}
	if resp.Error.Code != "not_found" {
		t.Errorf("got code %q, want %q", resp.Error.Code, "not_found")
	}
	if resp.Error.NextActionHint != "check the run ID and retry" {
		t.Errorf("got next_action_hint %q, want %q", resp.Error.NextActionHint, "check the run ID and retry")
	}
}
