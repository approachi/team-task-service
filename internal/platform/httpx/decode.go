package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/zhuk/team-task-service/internal/apperr"
)

func DecodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return apperr.Validation("body", "invalid JSON: "+err.Error())
	}
	return nil
}
