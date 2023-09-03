package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Crocmagnon/greenlight/internal/jsonlog"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     log.New(app.logger, "", 0),
	}

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.PrintInfo("caught signal", jsonlog.Properties{"signal": s.String()})

		os.Exit(0)
	}()

	app.logger.PrintInfo("starting server", jsonlog.Properties{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	err := srv.ListenAndServe()
	if err != nil {
		return fmt.Errorf("serving http: %w", err)
	}

	return nil
}
