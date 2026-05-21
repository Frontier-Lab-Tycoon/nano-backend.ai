package projectserv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/dto/response"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

func TestNewProjectHandlerRequiresRunService(t *testing.T) {
	_, err := newProjectHandler(Args{})
	if err == nil {
		t.Fatal("got nil error, want dependency error")
	}
}

func TestListRunsReturnsEnvelope(t *testing.T) {
	fixture := newProjectHandlerFixture(t)
	runID := fixture.givenRun()

	rec := fixture.listRuns(fixture.projectID.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Data response.ProjectRunsData `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Limit != defaultProjectRunsLimit {
		t.Fatalf("got limit %d, want %d", body.Data.Limit, defaultProjectRunsLimit)
	}
	if len(body.Data.Runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(body.Data.Runs))
	}
	if body.Data.Runs[0].ID != runID {
		t.Fatalf("got run id %s, want %s", body.Data.Runs[0].ID, runID)
	}
	if body.Data.Runs[0].ProjectID != fixture.projectID {
		t.Fatalf("got project id %s, want %s", body.Data.Runs[0].ProjectID, fixture.projectID)
	}
}

func TestListRunsReturnsEmptyProjectEnvelope(t *testing.T) {
	fixture := newProjectHandlerFixture(t)

	rec := fixture.listRuns(fixture.projectID.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Data response.ProjectRunsData `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Runs == nil {
		t.Fatal("got nil runs, want empty array")
	}
	if len(body.Data.Runs) != 0 {
		t.Fatalf("got %d runs, want 0", len(body.Data.Runs))
	}
}

func TestListRunsReturnsInvalidProjectIDEnvelope(t *testing.T) {
	fixture := newProjectHandlerFixture(t)

	rec := fixture.listRuns("not-a-uuid")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body response.Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error == nil || body.Error.Code != "invalid_input" {
		t.Fatalf("got error payload %#v, want code invalid_input", body.Error)
	}
}

func TestListRunsMapsNotFound(t *testing.T) {
	fixture := newProjectHandlerFixture(t)
	fixture.givenServiceError(errordef.ErrNotFound)

	rec := fixture.listRuns(fixture.projectID.String())

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

type projectHandlerFixture struct {
	t         *testing.T
	projectID uuid.UUID
	svc       *stubRunService
	handler   *projectHandler
}

func newProjectHandlerFixture(t *testing.T) *projectHandlerFixture {
	t.Helper()
	svc := &stubRunService{}
	return &projectHandlerFixture{
		t:         t,
		projectID: uuid.New(),
		svc:       svc,
		handler:   &projectHandler{svc: svc},
	}
}

func (f *projectHandlerFixture) givenRun() uuid.UUID {
	f.t.Helper()
	runID := uuid.New()
	f.svc.runs = append(f.svc.runs, run.Run{
		ID:     runID,
		SpecID: uuid.New(),
		Lifecycle: run.Lifecycle{
			Status:    run.Queued,
			CreatedAt: time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC),
		},
	})
	return runID
}

func (f *projectHandlerFixture) givenServiceError(err error) {
	f.t.Helper()
	f.svc.err = err
}

func (f *projectHandlerFixture) listRuns(id string) *httptest.ResponseRecorder {
	f.t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+id+"/runs", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	if err := f.handler.listRuns(c); err != nil {
		f.t.Fatalf("list runs: %v", err)
	}
	return rec
}

type stubRunService struct {
	runs []run.Run
	err  error
}

func (s *stubRunService) ListProjectRuns(_ context.Context, _ uuid.UUID, _ int) ([]run.Run, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.runs == nil {
		return []run.Run{}, nil
	}
	return s.runs, nil
}
