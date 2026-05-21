package runserv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/common/dto/response"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

func TestNewRunHandlerRequiresRunService(t *testing.T) {
	_, err := newRunHandler(Args{})
	if err == nil {
		t.Fatal("got nil error, want dependency error")
	}
}

func TestGetSpecReturnsEnvelope(t *testing.T) {
	runID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	runSpec := spec.Spec{
		ID:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		ProjectID: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Name:      "mergeowl-exp-42",
	}
	handler := &runHandler{svc: &stubRunService{spec: runSpec}}

	rec := performGetSpec(t, handler, runID.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Status int       `json:"status"`
		Data   spec.Spec `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != http.StatusOK {
		t.Fatalf("got body status %d, want %d", body.Status, http.StatusOK)
	}
	if body.Data.ID != runSpec.ID {
		t.Fatalf("got spec id %s, want %s", body.Data.ID, runSpec.ID)
	}
}

func TestGetSpecReturnsInvalidRunIDEnvelope(t *testing.T) {
	handler := &runHandler{svc: &stubRunService{}}

	rec := performGetSpec(t, handler, "not-a-uuid")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body response.Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error == nil || body.Error.Code != "invalid_run_id" {
		t.Fatalf("got error payload %#v, want code invalid_run_id", body.Error)
	}
}

func TestGetSpecMapsNotFound(t *testing.T) {
	runID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	handler := &runHandler{svc: &stubRunService{err: errordef.ErrNotFound}}

	rec := performGetSpec(t, handler, runID.String())

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func performGetSpec(t *testing.T, handler *runHandler, id string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+id+"/spec", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	if err := handler.getSpec(c); err != nil {
		t.Fatalf("get spec: %v", err)
	}
	return rec
}

type stubRunService struct {
	spec spec.Spec
	err  error
}

func (s *stubRunService) GetSpec(_ context.Context, _ uuid.UUID) (spec.Spec, error) {
	if s.err != nil {
		return spec.Spec{}, s.err
	}
	return s.spec, nil
}
