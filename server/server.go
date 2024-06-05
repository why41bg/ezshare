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

// Start 启动 HTTP 服务器
func Start(mux *mux.Router, address, cert, key string) error {
	srv := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	shutdown := make(chan error)
	// 启动一个 goroutine 来启动服务器，如果服务器启动失败，将错误信息发送到 shutdown 通道
	go func() {
		err := listenAndServe(srv, cert, key)
		shutdown <- err
	}()

	// 启动一个 goroutine 来监听操作系统的中断信号，如果接收到中断信号，就关闭服务器
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

	// 等待服务器启动失败或者接收到中断信号
	err := <-shutdown
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func listenAndServe(srv *http.Server, cert, key string) error {
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Error().Err(err).Msg("failed to create a TCP listener")
		return err
	}
	if cert != "" && key != "" {
		log.Debug().Msg("using TLS")
		log.Info().Msg("Started HTTP server")
		return srv.ServeTLS(listener, cert, key)
	} else {
		log.Debug().Msg("no TLS certificate provided, using plain HTTP")
		log.Info().Str("address", srv.Addr).Msg("Started HTTP server")
		return srv.Serve(listener)
	}
}
