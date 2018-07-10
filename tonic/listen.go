package tonic

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var defaultOpts = []ListenOptFunc{
	ListenAddr(":8080"),
	CatchSignals(os.Interrupt, syscall.SIGTERM),
	ShutdownTimeout(10 * time.Second),
	ReadHeaderTimeout(5 * time.Second),
	WriteTimeout(30 * time.Second),
	KeepAliveTimeout(90 * time.Second),
}

func ListenAndServe(handler http.Handler, errorHandler func(error), opt ...ListenOptFunc) {

	srv := &http.Server{Handler: handler}

	listenOpt := &ListenOpt{Server: srv}

	for _, o := range defaultOpts {
		o(listenOpt)
	}

	for _, o := range opt {
		o(listenOpt)
	}

	stop := make(chan struct{})

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed && errorHandler != nil {
			errorHandler(err)
		}
		close(stop)
	}()

	sig := make(chan os.Signal)

	if len(listenOpt.Signals) > 0 {
		signal.Notify(sig, listenOpt.Signals...)
	}

	select {
	case <-sig:
		ctx, cancel := context.WithTimeout(context.Background(), listenOpt.ShutdownTimeout)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil && errorHandler != nil {
			errorHandler(err)
		}

	case <-stop:
		break

	}
}

type ListenOpt struct {
	Server          *http.Server
	Signals         []os.Signal
	ShutdownTimeout time.Duration
}

type ListenOptFunc func(*ListenOpt)

func CatchSignals(sig ...os.Signal) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Signals = sig
	}
}

func ListenAddr(addr string) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Server.Addr = addr
	}
}

func ReadTimeout(t time.Duration) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Server.ReadTimeout = t
	}
}

func ReadHeaderTimeout(t time.Duration) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Server.ReadHeaderTimeout = t
	}
}

func WriteTimeout(t time.Duration) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Server.WriteTimeout = t
	}
}

func KeepAliveTimeout(t time.Duration) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.Server.IdleTimeout = t
	}
}

func ShutdownTimeout(t time.Duration) ListenOptFunc {
	return func(opt *ListenOpt) {
		opt.ShutdownTimeout = t
	}
}
