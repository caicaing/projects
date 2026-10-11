package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.woa.com/mooyliu/mooy_test/backend/api"
	"git.woa.com/mooyliu/mooy_test/backend/config"
	"git.woa.com/mooyliu/mooy_test/backend/storage/sqlite"
	taskflow "git.woa.com/mooyliu/mooy_test/framework"
)

func main() {
	if err := serve(); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func serve() (serveErr error) {
	path := flag.String("config", "conf.yaml", "后端配置文件")
	flag.Parse()
	settings, err := config.Load(*path)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", settings.Backend.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	initCtx, initCancel := context.WithTimeout(ctx, 15*time.Second)
	defer initCancel()
	store, err := sqlite.Open(initCtx, settings.Backend.Database)
	if err != nil {
		return err
	}
	engine, err := taskflow.New(initCtx, settings.EngineOptions(store),
		registerPlugins)
	if err != nil {
		return errors.Join(err, store.Close())
	}
	server := &http.Server{
		Handler:           api.New(engine, settings.Backend.Token),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second, WriteTimeout: 30 * time.Second,
		IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16,
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(),
			15*time.Second)
		defer cancel()
		engineErr := engine.Close(shutdownCtx)
		httpErr := server.Shutdown(shutdownCtx)
		var storeErr error
		if engineErr == nil {
			storeErr = store.Close()
		}
		serveErr = errors.Join(serveErr, engineErr, httpErr, storeErr)
	}()
	if err := engine.Start(ctx); err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	slog.Info("Taskflow 已启动", "address", listener.Addr().String())
	select {
	case <-ctx.Done():
	case err = <-errCh:
		if err == http.ErrServerClosed {
			err = nil
		}
	}
	return err
}
