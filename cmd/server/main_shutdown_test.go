package main

import (
	"net/http"
	"os"
	osSignal "os/signal"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestShutdownSignals(t *testing.T) {
	t.Cleanup(func() {
		signalNotify = osSignal.Notify
	})

	signalNotify = func(ch chan<- os.Signal, sig ...os.Signal) {
		go func() {
			ch <- syscall.SIGTERM
		}()
	}

	server := &http.Server{}
	called := make(chan struct{}, 1)
	server.RegisterOnShutdown(func() {
		called <- struct{}{}
	})

	logger := zaptest.NewLogger(t)
	shutdown(server, time.Millisecond, logger)

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatalf("expected server shutdown callback to execute")
	}
}
