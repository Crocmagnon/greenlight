package main

import (
	"net/http"
	"testing"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	conf := config{}
	conf.limiter.enabled = true
	conf.limiter.rps = 2
	conf.limiter.burst = 10 //nolint:revive
	conf.limiter.lastSeenMinutes = 1

	//nolint:revive
	tests := []struct {
		name            string
		overLimit       int
		expectedOK      int
		expectedTooMany int
	}{
		{"under limit", 0, 10, 0},
		{"over limit", 5, 10, 5},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApplication(t)
			app.config = conf
			server := newTestServer(t, app.routes())
			defer server.Close()

			res := map[int]int{}

			for i := 0; i < app.config.limiter.burst+test.overLimit; i++ {
				code, _ := server.get(t, "/v1/healthcheck")
				res[code]++
			}

			if len(res) > 2 {
				t.Errorf("got len(res) == %d, want <= %d", len(res), 2)
			}

			got := res[http.StatusOK]
			if got != test.expectedOK {
				t.Errorf("got %d responses with status %d, want %d", got, http.StatusOK, test.expectedOK)
			}

			got = res[http.StatusTooManyRequests]
			if got != test.expectedTooMany {
				t.Errorf("got %d responses with status %d, want %d", got, http.StatusTooManyRequests, test.expectedTooMany)
			}
		})
	}
}
