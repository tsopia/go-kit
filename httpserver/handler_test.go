package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Email string `json:"email"`
}

func (r loginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}

	return nil
}

type structuredValidationRequest struct {
	Email string `json:"email"`
}

func (r structuredValidationRequest) Validate() error {
	if r.Email != "" {
		return nil
	}

	return &ValidationError{
		Message: "request validation failed",
		Fields: []ValidationField{
			{
				Field:   "email",
				Message: "is required",
			},
		},
	}
}

type testHTTPError struct {
	status  int
	code    string
	message string
	details map[string]any
}

func (e *testHTTPError) Error() string {
	return e.message
}

func (e *testHTTPError) StatusCode() int {
	return e.status
}

func (e *testHTTPError) ErrorCode() string {
	return e.code
}

func (e *testHTTPError) ErrorMessage() string {
	return e.message
}

func (e *testHTTPError) ErrorDetails() map[string]any {
	return e.details
}

func TestHandleJSONSuccess(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["email"] != "foo@example.com" {
		t.Fatalf("expected email to round-trip, got %q", resp["email"])
	}
}

func TestHandleJSONValidateError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "validation_failed" {
		t.Fatalf("expected error code %q, got %q", "validation_failed", resp.Code)
	}
	if resp.Message != "email is required" {
		t.Fatalf("expected message %q, got %q", "email is required", resp.Message)
	}
	if resp.Details != nil {
		t.Fatalf("expected no details, got %#v", resp.Details)
	}
}

func TestHandleJSONDecodeError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", resp.Code)
	}
	if !strings.Contains(resp.Message, "decode request") {
		t.Fatalf("expected message to contain decode context, got %q", resp.Message)
	}
	if resp.Details != nil {
		t.Fatalf("expected no details, got %#v", resp.Details)
	}
}

func TestHandleJSONDefaultInternalErrorResponse(t *testing.T) {
	sentinel := errors.New("database unavailable")

	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return nil, sentinel
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "internal_error" {
		t.Fatalf("expected error code %q, got %q", "internal_error", resp.Code)
	}
	if resp.Message != sentinel.Error() {
		t.Fatalf("expected message %q, got %q", sentinel.Error(), resp.Message)
	}
	if resp.Details != nil {
		t.Fatalf("expected no details, got %#v", resp.Details)
	}
}

func TestHandleJSONStructuredValidationError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/register", HandleJSON(func(ctx context.Context, req structuredValidationRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/register", gin.H{"email": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "validation_failed" {
		t.Fatalf("expected error code %q, got %q", "validation_failed", resp.Code)
	}
	if resp.Message != "request validation failed" {
		t.Fatalf("expected message %q, got %q", "request validation failed", resp.Message)
	}

	fields := extractDetailFields(t, resp.Details)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(fields))
	}
	if fields[0]["field"] != "email" {
		t.Fatalf("expected field %q, got %#v", "email", fields[0]["field"])
	}
	if fields[0]["message"] != "is required" {
		t.Fatalf("expected field message %q, got %#v", "is required", fields[0]["message"])
	}
}

func TestHandleJSONWithValidators(t *testing.T) {
	type registerRequest struct {
		Email string `json:"email"`
	}

	srv := NewServer(nil)
	srv.POST("/register", HandleJSON(
		func(ctx context.Context, req registerRequest) (gin.H, error) {
			return gin.H{"email": req.Email}, nil
		},
		WithValidators(func(ctx context.Context, req registerRequest) error {
			if strings.HasSuffix(req.Email, "@company.com") {
				return nil
			}

			return &ValidationError{
				Message: "request validation failed",
				Fields: []ValidationField{
					{
						Field:   "email",
						Message: "must use company email",
					},
				},
			}
		}),
	))

	w := performJSONRequest(t, srv, http.MethodPost, "/register", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "validation_failed" {
		t.Fatalf("expected error code %q, got %q", "validation_failed", resp.Code)
	}
	fields := extractDetailFields(t, resp.Details)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(fields))
	}
	if fields[0]["message"] != "must use company email" {
		t.Fatalf("expected field message %q, got %#v", "must use company email", fields[0]["message"])
	}
}

func TestHandleHTTPError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return nil, &testHTTPError{
			status:  http.StatusConflict,
			code:    "user_conflict",
			message: "user already exists",
			details: map[string]any{
				"resource": "user",
			},
		}
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.Code != "user_conflict" {
		t.Fatalf("expected error code %q, got %q", "user_conflict", resp.Code)
	}
	if resp.Message != "user already exists" {
		t.Fatalf("expected message %q, got %q", "user already exists", resp.Message)
	}
	if resp.Details["resource"] != "user" {
		t.Fatalf("expected resource detail %q, got %#v", "user", resp.Details["resource"])
	}
}

func TestHandleQueryShortcut(t *testing.T) {
	type listRequest struct {
		Page int `form:"page"`
	}

	srv := NewServer(nil)
	srv.GET("/users", HandleQuery(func(ctx context.Context, req listRequest) (gin.H, error) {
		return gin.H{"page": req.Page}, nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?page=2", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"page":2`)) {
		t.Fatalf("expected query shortcut to populate page, got %q", w.Body.String())
	}
}

func TestHandleURIShortcut(t *testing.T) {
	type getUserRequest struct {
		ID string `uri:"id"`
	}

	srv := NewServer(nil)
	srv.GET("/users/:id", HandleURI(func(ctx context.Context, req getUserRequest) (gin.H, error) {
		return gin.H{"id": req.ID}, nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"123"`)) {
		t.Fatalf("expected uri shortcut to populate id, got %q", w.Body.String())
	}
}

func TestHandleQueryURIShortcut(t *testing.T) {
	type request struct {
		ID      string `uri:"id"`
		Verbose bool   `form:"verbose"`
	}

	srv := NewServer(nil)
	srv.GET("/users/:id", HandleQueryURI(func(ctx context.Context, req request) (gin.H, error) {
		return gin.H{
			"id":      req.ID,
			"verbose": req.Verbose,
		}, nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/abc?verbose=true", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"abc"`)) {
		t.Fatalf("expected query+uri shortcut to populate id, got %q", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"verbose":true`)) {
		t.Fatalf("expected query+uri shortcut to populate verbose, got %q", w.Body.String())
	}
}

func TestHandleJSONCustomErrorMapper(t *testing.T) {
	sentinel := errors.New("duplicate user")

	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(
		func(ctx context.Context, req loginRequest) (gin.H, error) {
			return nil, sentinel
		},
		WithErrorMapper(func(err error) (int, any) {
			if errors.Is(err, sentinel) {
				return http.StatusConflict, gin.H{"error": "duplicate"}
			}

			return http.StatusInternalServerError, gin.H{"error": err.Error()}
		}),
	))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleJSONCustomEncoderAndSuccessStatus(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(
		func(ctx context.Context, req loginRequest) (string, error) {
			return "token-123", nil
		},
		WithSuccessStatus(http.StatusAccepted),
		WithEncoder(func(c *gin.Context, status int, resp any) {
			token, _ := resp.(string)
			c.String(status, "token=%s", token)
		}),
	))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
	if w.Body.String() != "token=token-123" {
		t.Fatalf("expected custom encoder body, got %q", w.Body.String())
	}
}

func TestHandleQuerySuccess(t *testing.T) {
	type listRequest struct {
		Page int `form:"page"`
	}

	srv := NewServer(nil)
	srv.GET("/users", Handle(
		func(ctx context.Context, req listRequest) (gin.H, error) {
			return gin.H{"page": req.Page}, nil
		},
		WithDecoder(DecodeQuery[listRequest]()),
	))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?page=2", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"page":2`)) {
		t.Fatalf("expected query decoder to populate page, got %q", w.Body.String())
	}
}

func TestHandleURISuccess(t *testing.T) {
	type getUserRequest struct {
		ID string `uri:"id"`
	}

	srv := NewServer(nil)
	srv.GET("/users/:id", Handle(
		func(ctx context.Context, req getUserRequest) (gin.H, error) {
			return gin.H{"id": req.ID}, nil
		},
		WithDecoder(DecodeURI[getUserRequest]()),
	))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"123"`)) {
		t.Fatalf("expected uri decoder to populate id, got %q", w.Body.String())
	}
}

func TestComposeDecoder(t *testing.T) {
	type request struct {
		ID      string `uri:"id"`
		Verbose bool   `form:"verbose"`
	}

	srv := NewServer(nil)
	srv.GET("/users/:id", Handle(
		func(ctx context.Context, req request) (gin.H, error) {
			return gin.H{
				"id":      req.ID,
				"verbose": req.Verbose,
			}, nil
		},
		WithDecoder(ComposeDecoder(
			DecodeURI[request](),
			DecodeQuery[request](),
		)),
	))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/abc?verbose=true", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"abc"`)) {
		t.Fatalf("expected composed decoder to populate id, got %q", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"verbose":true`)) {
		t.Fatalf("expected composed decoder to populate verbose, got %q", w.Body.String())
	}
}

func TestTypedAndNativeHandlersCoexist(t *testing.T) {
	type listRequest struct {
		Page int `form:"page"`
	}

	srv := NewServer(nil)
	srv.GET("/users", Handle(
		func(ctx context.Context, req listRequest) (gin.H, error) {
			return gin.H{"page": req.Page}, nil
		},
		WithDecoder(DecodeQuery[listRequest]()),
	))
	srv.GET("/native", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"mode": "native"})
	})

	typedResp := httptest.NewRecorder()
	typedReq := httptest.NewRequest(http.MethodGet, "/users?page=3", nil)
	srv.Engine().ServeHTTP(typedResp, typedReq)

	if typedResp.Code != http.StatusOK {
		t.Fatalf("expected typed handler status %d, got %d", http.StatusOK, typedResp.Code)
	}

	nativeResp := httptest.NewRecorder()
	nativeReq := httptest.NewRequest(http.MethodGet, "/native", nil)
	srv.Engine().ServeHTTP(nativeResp, nativeReq)

	if nativeResp.Code != http.StatusOK {
		t.Fatalf("expected native handler status %d, got %d", http.StatusOK, nativeResp.Code)
	}
	if !bytes.Contains(nativeResp.Body.Bytes(), []byte(`"mode":"native"`)) {
		t.Fatalf("expected native handler body, got %q", nativeResp.Body.String())
	}
}

func performJSONRequest(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)

	return w
}

type errorResponsePayload struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func decodeErrorResponse(t *testing.T, w *httptest.ResponseRecorder) errorResponsePayload {
	t.Helper()

	var resp errorResponsePayload
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	return resp
}

func extractDetailFields(t *testing.T, details map[string]any) []map[string]any {
	t.Helper()

	if details == nil {
		t.Fatal("expected details to be present")
	}

	rawFields, ok := details["fields"]
	if !ok {
		t.Fatalf("expected details.fields to exist, got %#v", details)
	}

	fieldList, ok := rawFields.([]any)
	if !ok {
		t.Fatalf("expected details.fields to be a slice, got %T", rawFields)
	}

	fields := make([]map[string]any, 0, len(fieldList))
	for _, item := range fieldList {
		field, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected field item to be an object, got %T", item)
		}
		fields = append(fields, field)
	}

	return fields
}
