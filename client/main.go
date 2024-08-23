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
	serverAddr      = "localhost:8080"
	timeout         = 5 * time.Second
	readBufferSize  = 256
	shutdownTimeout = 3 * time.Second
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var conn net.Conn
	respCh := make(chan []byte, 1)

	retryConfig := retry.Config{
		MaxAttempts:      10,
		SleepCoefficient: 1.5,
		StartSleep:       250 * time.Millisecond,
	}

	dialRetry := retry.NewFunc(retryConfig, func(ctx context.Context) error {
		var err error
		conn, err = net.Dial("tcp", serverAddr)
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	})

	getServerResponseRetry := retry.NewFunc(retryConfig, func(ctx context.Context) error {
		err := GetServerResponse(conn, respCh)
		if err != nil {
			log.Printf("get server response: %v\n", err)
			return err
		}

		return nil
	})

	errCh := make(chan error, 1)
	go func() {
		err := dialRetry(ctx)
		if err != nil {
			errCh <- err
			return
		}

		err = getServerResponseRetry(ctx)
		if err != nil {
			errCh <- err
			return
		}
	}()

	var respBuf []byte
	select {
	case <-ctx.Done():
		log.Println("quitting before getting response..")
	case respBuf = <-respCh:
		log.Println("got response from server")
	case err := <-errCh:
		log.Printf("error while interacting with server: %v\n", err)
	}

	log.Println("starting graceful shutdown")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	closed := make(chan struct{}, 1)
	go func() {
		if conn != nil {
			conn.Close()
		}
		closed <- struct{}{}
	}()

	select {
	case <-shutdownCtx.Done():
		log.Printf("listener close: %v\n", shutdownCtx.Err())
	case <-closed:
		log.Println("finished")
	}

	resp := string(respBuf)
	log.Printf("got resp: %q\n", resp)
	if resp == "OK" {
		log.Println("client successfully get OK")
	} else {
		log.Println("client didn't get OK")
	}
}

func GetServerResponse(conn net.Conn, respCh chan<- []byte) error {
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, readBufferSize)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("in get server response: %v\n", err)
		}

		respCh <- buf[:n]
		close(respCh)
	}()

	return nil
}
