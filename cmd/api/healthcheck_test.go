package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	server := newTestServer(t, app.routes())
	defer server.Close()

	code, body := server.get(t, "/v1/healthcheck")

	if code != http.StatusOK {
		t.Errorf("got http status %d want 200", code)
	}

	if !strings.Contains(body, "available") {
		t.Errorf("expected body %q to contain %q", body, "available")
	}
}
