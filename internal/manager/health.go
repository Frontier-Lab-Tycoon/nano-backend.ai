package manager

import (
	"encoding/json"
	"net/http"

	"github.com/seedspirit/nano-backend.ai/internal/common/response"
)

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	resp := response.OK("healthy", "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck // best-effort response write
}
