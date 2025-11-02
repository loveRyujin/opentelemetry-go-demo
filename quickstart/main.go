package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	ctx := context.Background()
	serviceName := "dice"
	serviceVersion := "0.1.0"
	otelShutdown, err := setupOTelSDK(ctx, serviceName, serviceVersion)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, otelShutdown(ctx))
	}()

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		log.Print("Starting to listen localhost:8080...")
		if srvErr := server.ListenAndServe(); srvErr != nil {
			log.Printf("server error: %v", srvErr)
		}
	}()

	<-quit

	if err = server.Shutdown(ctx); err != nil {
		return err
	}

	return
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	handleFunc := func(pattern string, handlerFunc func(w http.ResponseWriter, r *http.Request)) {
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	handleFunc("/rolldice", rolldice)

	handler := otelhttp.NewHandler(mux, "/")
	return handler
}
