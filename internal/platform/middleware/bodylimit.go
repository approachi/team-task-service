package middleware

import "net/http"

// MaxRequestBodyBytes caps request bodies the JSON decoder will read. Chosen
// well above any legitimate payload (task/team/auth requests are all small,
// structured JSON) but small enough that a client can't force large memory
// allocations by sending an oversized body.
const MaxRequestBodyBytes = 1 << 20 // 1 MiB

// LimitBody wraps every request body in http.MaxBytesReader so a decoder
// reading past the limit gets an error instead of the server allocating
// unbounded memory for an oversized payload. Mounted before auth so it
// covers /register and /login too.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}
