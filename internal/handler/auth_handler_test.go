package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/service"
)

func newAuthTestRouter() (http.Handler, *fakeUserRepo) {
	users := newFakeUserRepo()
	authSvc := service.NewAuthService(users, []byte(testJWTSecret), time.Hour)
	h := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	return r, users
}

func TestAuthHandler_Register(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		body := `{"email":"new@example.com","password":"supersecret","name":"New User"}`
		rec := do(router, newRequest(http.MethodPost, "/register", body))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var env dataEnvelope[dto.UserResponse]
		decodeInto(t, rec, &env)
		if env.Data.Email != "new@example.com" || env.Data.Name != "New User" || env.Data.ID == 0 {
			t.Fatalf("unexpected user in response: %+v", env.Data)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		rec := do(router, newRequest(http.MethodPost, "/register", `{"email":`))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		body := `{"email":"new@example.com","password":"short","name":"New User"}`
		rec := do(router, newRequest(http.MethodPost, "/register", body))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		env := decodeError(t, rec)
		if _, ok := env.Error.Details["password"]; !ok {
			t.Fatalf("expected password validation detail, got %+v", env.Error.Details)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		router, users := newAuthTestRouter()
		users.seed("dup@example.com", "Existing User")

		body := `{"email":"dup@example.com","password":"supersecret","name":"New User"}`
		rec := do(router, newRequest(http.MethodPost, "/register", body))

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409 (body=%s)", rec.Code, rec.Body.String())
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	registerUser := func(t *testing.T, router http.Handler, email, password string) {
		t.Helper()
		body := `{"email":"` + email + `","password":"` + password + `","name":"Test User"}`
		rec := do(router, newRequest(http.MethodPost, "/register", body))
		if rec.Code != http.StatusCreated {
			t.Fatalf("setup: register failed with status %d (body=%s)", rec.Code, rec.Body.String())
		}
	}

	t.Run("success", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		registerUser(t, router, "login@example.com", "supersecret")

		rec := do(router, newRequest(http.MethodPost, "/login", `{"email":"login@example.com","password":"supersecret"}`))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[dto.LoginResponse]
		decodeInto(t, rec, &env)
		if env.Data.Token == "" {
			t.Fatal("expected a non-empty token")
		}
		if env.Data.User.Email != "login@example.com" {
			t.Fatalf("unexpected user in login response: %+v", env.Data.User)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		rec := do(router, newRequest(http.MethodPost, "/login", `not json`))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure: empty password", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		rec := do(router, newRequest(http.MethodPost, "/login", `{"email":"a@example.com","password":""}`))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		registerUser(t, router, "wrongpw@example.com", "correcthorse")

		rec := do(router, newRequest(http.MethodPost, "/login", `{"email":"wrongpw@example.com","password":"incorrect"}`))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("unknown email returns the same error as wrong password", func(t *testing.T) {
		router, _ := newAuthTestRouter()
		registerUser(t, router, "known@example.com", "correcthorse")

		wrongPwRec := do(router, newRequest(http.MethodPost, "/login", `{"email":"known@example.com","password":"incorrect"}`))
		unknownRec := do(router, newRequest(http.MethodPost, "/login", `{"email":"unknown@example.com","password":"incorrect"}`))

		if unknownRec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (body=%s)", unknownRec.Code, unknownRec.Body.String())
		}
		wrongPwErr := decodeError(t, wrongPwRec)
		unknownErr := decodeError(t, unknownRec)
		if wrongPwErr.Error.Message != unknownErr.Error.Message || wrongPwErr.Error.Code != unknownErr.Error.Code {
			t.Fatalf("unknown-email error %+v differs from wrong-password error %+v; this is an enumeration leak", unknownErr.Error, wrongPwErr.Error)
		}
	})
}
