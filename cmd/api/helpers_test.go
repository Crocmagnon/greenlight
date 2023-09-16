package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Crocmagnon/greenlight/internal/data"
	"github.com/Crocmagnon/greenlight/internal/mailer"
	"github.com/jmoiron/sqlx"
)

func newTestApplication(tb testing.TB) *application {
	tb.Helper()

	db := sqlx.DB{}

	return &application{
		config: config{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		models: data.NewModels(&db),
		mailer: mailer.Mailer{},
		wg:     sync.WaitGroup{},
	}
}

type testServer struct {
	*httptest.Server
}

func newTestServer(tb testing.TB, routes http.Handler) *testServer {
	tb.Helper()

	srv := httptest.NewTLSServer(routes)

	tb.Cleanup(func() {
		srv.Close()
	})

	return &testServer{srv}
}

func (ts *testServer) get(tb testing.TB, urlPath string) (int, string) {
	tb.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+urlPath, nil)
	if err != nil {
		tb.Fatal(err)
	}

	res, err := ts.Client().Do(req)
	if err != nil {
		tb.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		tb.Fatal(err)
	}

	return res.StatusCode, string(body)
}
