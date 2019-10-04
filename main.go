package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

const COUNT_OF_REQ = 2000
const TIMEOUT_MS = 5000

func getGooglePage(ctx context.Context, waiter chan int) {
	resCh := make(chan *http.Response)
	errCh := make(chan error)

	go func() {
		resp, err := http.Get("https://google.com")

		if err != nil {
			errCh <- err
			return
		}

		defer func() {
			resp.Body.Close()
			resCh <- resp
		}()

		ioutil.ReadAll(resp.Body)
	}()

	select {
	case <-errCh:
		return
	case <-resCh:
		waiter <- 0
	case <-ctx.Done():
		return
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	waiter := make(chan int, 100)
	done := make(chan bool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := int32(0)

	start := time.Now()

	time.AfterFunc(TIMEOUT_MS*time.Millisecond, func() { done <- true })

	for i := 0; i < COUNT_OF_REQ; i++ {
		_ = i
		go getGooglePage(ctx, waiter)
	}

	for {
		if atomic.LoadInt32(&count) == COUNT_OF_REQ {
			fmt.Println("100%, finished.")
			return
		}

		select {
		case <-done:
			elapsed := time.Since(start)

			close(waiter)
			ctx.Done()

			fmt.Println(
				fmt.Sprintf("Count of request before cancel %d: ", atomic.LoadInt32(&count)))

			fmt.Println("Time spent:", elapsed.String())

			return
		case <-waiter:
			fmt.Println(
				fmt.Sprintf("%.2f%%", float32(count)/float32(COUNT_OF_REQ)*float32(100)))
			atomic.AddInt32(&count, 1)
		}
	}
}
