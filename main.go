package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	probing "github.com/prometheus-community/pro-bing"
)

type pingResult struct {
	StartTime time.Time
	FromHost  string
	ToHost    string
	LatencyMS float32
	Timeout   bool
	// Interface // TODO: Network interface
}

var err error

func main() {

	connStr := ""
	if os.Getenv("PING_TIMESCALEDB_CONNSTR") != "" {
		connStr = os.Getenv("PING_TIMESCALEDB_CONNSTR")
	} else {
		log.Fatalf("Environment variable PING_TIMESCALEDB_CONNSTR must be set")
	}

	interval := time.Duration(200) * time.Millisecond
	if os.Getenv("PING_INTERVAL") != "" {
		interval, err = time.ParseDuration(os.Getenv("PING_INTERVAL"))
		if err != nil {
			log.Fatalf("Could not parse PING_INTERVAL: %v", err)
		}
	}

	timeout := time.Duration(2000) * time.Millisecond
	if os.Getenv("PING_TIMEOUT") != "" {
		timeout, err = time.ParseDuration(os.Getenv("PING_TIMEOUT"))
		if err != nil {
			log.Fatalf("Could not parse PING_TIMEOUT: %v", err)
		}
	}

	destinations := []string{"www.vg.no", "192.168.1.1"}
	if os.Getenv("PING_DESTINATIONS") != "" {
		destinations = strings.Split(os.Getenv("PING_DESTINATIONS"), ",")
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get own hostname: %v", err)
	}

	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	var dbCheck string
	err = dbpool.QueryRow(ctx, "select 'Database connected.'").Scan(&dbCheck)
	if err != nil {
		log.Fatalf("Unable to query database: %v\n", err)
	}
	fmt.Println(dbCheck)

	results := make(chan pingResult, 1000)

	// Collect results in a separate goroutine.
	go func() {
		for {
			result := <-results

			query := `INSERT INTO pings (time, from_host, to_host, interface, latency_ms, error, timeout) VALUES ($1, $2, $3, $4, $5, $6, $7);`

			_, err := dbpool.Exec(ctx, query, result.StartTime, result.FromHost, result.ToHost, "", result.LatencyMS, "", result.Timeout)
			if err != nil {
				log.Fatalf("Unable to insert data into database: %v\n", err)
			}
		}
	}()

	for _, destination := range destinations {
		// Start pinging each destination separately.
		go func(destination string) {
			ticker := time.NewTicker(interval)
			for range ticker.C {
				results <- ping(hostname, destination, timeout)
			}
		}(destination)
	}

	go func() {
		for {
			fmt.Printf("Result processing queue size (0 is good, 1000 is bad): %d\n", len(results))
			time.Sleep(time.Second * 5)
		}
	}()

	// Blocks forever to keep program running
	select {}
}

func ping(fromHost string, toHost string, timeout time.Duration) (result pingResult) {
	result.FromHost = fromHost
	result.ToHost = toHost

	pinger, err := probing.NewPinger(toHost)
	if err != nil {
		panic(err)
	}
	pinger.Count = 1
	pinger.Timeout = timeout

	result.StartTime = time.Now()
	err = pinger.Run()
	if err != nil {
		panic(err)
	}
	stats := pinger.Statistics()

	result.LatencyMS = float32(stats.MaxRtt) / float32(time.Millisecond)

	if stats.PacketLoss > 0 {
		result.Timeout = true
		result.LatencyMS = float32(timeout) / float32(time.Millisecond)
	}

	return result
}
