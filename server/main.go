package main

import (
	"context"
	"github.com/Kugeki/p_hw1/retry"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"
)

const (
	addr            = "localhost:8080"
	shutdownTimeout = 3 * time.Second
	message         = "OK"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	RunListening(ctx, listener)

	<-ctx.Done()
	log.Println("starting graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	closed := make(chan struct{}, 1)
	go func() {
		listener.Close()
		closed <- struct{}{}
	}()

	select {
	case <-shutdownCtx.Done():
		log.Printf("listener close: %v\n", shutdownCtx.Err())
	case <-closed:
		log.Println("finished")
	}
}

func RunListening(ctx context.Context, listener net.Listener) {
	log.Printf("started listening on %v...\n", listener.Addr())

	go func() {
		retryConfig := retry.Config{
			MaxAttempts:      10,
			SleepCoefficient: 1.5,
			StartSleep:       250 * time.Millisecond,
		}

		acceptWithRetry := retry.NewFunc(retryConfig, func(ctx context.Context) error {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("listener accept: %v", err)
				return err
			}

			log.Println("new connection!")

			go HandleConn(ctx, conn)
			return nil
		})

		for {
			err := acceptWithRetry(ctx)
			if err != nil {
				log.Fatalf("in run listening: %v\n", err)
			}
		}
	}()
}

func HandleConn(ctx context.Context, conn net.Conn) {
	_, err := conn.Write([]byte(message))
	if err != nil {
		log.Printf("conn write: %v", err)
	}

	log.Println("connection successfully handled!")

	conn.Close()
}
