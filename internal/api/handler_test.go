package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReceiveEndpoint(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/receive", strings.NewReader(`{"sessionId":"s-1","stage":"begin"}`))
	Handler{}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"sessionId":"s-1"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
