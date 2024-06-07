package server

import (
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// Start starts the http/https server.
func Start(mux *mux.Router, address, cert, key string) error {
	srv := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	shutdown := make(chan error)
	go func() {
		err := listenAndServe(srv, cert, key)
		shutdown <- err
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		log.Info().Msg("Received interrupt signal, shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			// 如果没有 err 返回，说明服务器已经被优雅关闭
			shutdown <- err
		}
	}()

	err := <-shutdown
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// listenAndServe starts the server.
func listenAndServe(srv *http.Server, cert, key string) error {
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a TCP listener")
		return err
	}
	if cert != "" && key != "" {
		log.Debug().Str("address", srv.Addr).Msg("Started HTTPS server")
		return srv.ServeTLS(listener, cert, key)
	} else {
		log.Debug().Str("address", srv.Addr).Msg("Started HTTP server")
		return srv.Serve(listener)
	}
}
