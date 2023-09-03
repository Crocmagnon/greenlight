package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Crocmagnon/greenlight/internal/jsonlog"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err)) //nolint:goerr113
			}
		}()

		next.ServeHTTP(w, r)
	})
}

//nolint:gocognit
func (app *application) rateLimit(next http.Handler) http.Handler {
	if !app.config.limiter.enabled {
		return next
	}

	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	ticker := time.NewTicker(time.Minute)

	go func() {
		for range ticker.C {
			app.logger.PrintInfo("ticking", nil)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > time.Duration(app.config.limiter.lastSeenMinutes)*time.Minute {
					app.logger.PrintInfo("deleting ip", jsonlog.Properties{"ip": ip})
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
