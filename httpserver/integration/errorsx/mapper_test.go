package errorsx

import (
	"bytes"
	stderrors "errors"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	kiterrors "github.com/tsopia/go-kit/errors"
	"github.com/tsopia/go-kit/httpserver"
)

func TestResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   ResponseBody
	}{
		{
			name:       "invalid param",
			err:        kiterrors.InvalidParam.New("email is required"),
			wantStatus: http.StatusBadRequest,
			wantBody: ResponseBody{
				Code:    kiterrors.InvalidParam.Code,
				Name:    kiterrors.InvalidParam.Name,
				Message: "email is required",
			},
		},
		{
			name:       "not found",
			err:        kiterrors.NotFound.New("user not found"),
			wantStatus: http.StatusNotFound,
			wantBody: ResponseBody{
				Code:    kiterrors.NotFound.Code,
				Name:    kiterrors.NotFound.Name,
				Message: "user not found",
			},
		},
		{
			name:       "plain error falls back to internal",
			err:        stderrors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantBody: ResponseBody{
				Code:    kiterrors.Internal.Code,
				Name:    kiterrors.Internal.Name,
				Message: "internal server error",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotStatus, gotBody := Response(tc.err)
			if gotStatus != tc.wantStatus {
				t.Fatalf("Response(%v) status = %d, want %d", tc.err, gotStatus, tc.wantStatus)
			}
			if !reflect.DeepEqual(gotBody, tc.wantBody) {
				t.Fatalf("Response(%v) body = %#v, want %#v", tc.err, gotBody, tc.wantBody)
			}
		})
	}
}

func TestMapper(t *testing.T) {
	t.Parallel()

	type createUserRequest struct {
		Email string `json:"email"`
	}

	srv := httpserver.NewServer(nil)
	srv.POST("/users", httpserver.HandleJSON(
		func(ctx context.Context, req createUserRequest) (gin.H, error) {
			return nil, kiterrors.InvalidParam.New("email is required")
		},
		httpserver.WithErrorMapper(Mapper()),
	))

	payload, err := json.Marshal(createUserRequest{Email: "foo@example.com"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var body ResponseBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	want := ResponseBody{
		Code:    kiterrors.InvalidParam.Code,
		Name:    kiterrors.InvalidParam.Name,
		Message: "email is required",
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("response body = %#v, want %#v", body, want)
	}
}
