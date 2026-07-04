package httpx

import (
	"net/http"
	"strconv"

	"github.com/zhuk/team-task-service/internal/apperr"
)

type PageRequest struct {
	Page     int
	PageSize int
}

func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ParsePageRequest reads "page"/"page_size" query params, applying
// defaultSize when page_size is absent and capping it at maxSize.
func ParsePageRequest(r *http.Request, defaultSize, maxSize int) (PageRequest, error) {
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return PageRequest{}, apperr.Validation("page", "must be a positive integer")
		}
		page = n
	}

	pageSize := defaultSize
	if v := r.URL.Query().Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return PageRequest{}, apperr.Validation("page_size", "must be a positive integer")
		}
		pageSize = n
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}

	return PageRequest{Page: page, PageSize: pageSize}, nil
}

func NewMeta(p PageRequest, total int) Meta {
	return Meta{Page: p.Page, PageSize: p.PageSize, Total: total}
}
