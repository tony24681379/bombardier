package lib

import (
	"flag"
	"runtime"
	"testing"
	"time"
)

var (
	serverPort = flag.String("port", "8080", "port to use for benchmarks")
	clientType = flag.String("client-type", "fasthttp",
		"client to use in benchmarks")
)

var (
	longDuration = 9001 * time.Hour
	highRate     = uint64(1000000)
)

func BenchmarkBombardierSingleReqPerf(b *testing.B) {
	addr := "localhost:" + *serverPort
	benchmarkFireRequest(Config{
		NumConns:       defaultNumberOfConns,
		NumReqs:        nil,
		Duration:       &longDuration,
		Url:            "http://" + addr,
		Headers:        new(HeadersList),
		Timeout:        defaultTimeout,
		Method:         "GET",
		Body:           "",
		PrintLatencies: false,
		ClientType:     clientTypeFromString(*clientType),
	}, b)
}

func BenchmarkBombardierRateLimitPerf(b *testing.B) {
	addr := "localhost:" + *serverPort
	benchmarkFireRequest(Config{
		NumConns:       defaultNumberOfConns,
		NumReqs:        nil,
		Duration:       &longDuration,
		Url:            "http://" + addr,
		Headers:        new(HeadersList),
		Timeout:        defaultTimeout,
		Method:         "GET",
		Body:           "",
		PrintLatencies: false,
		Rate:           &highRate,
		ClientType:     clientTypeFromString(*clientType),
	}, b)
}

func benchmarkFireRequest(c Config, bm *testing.B) {
	b, e := NewBombardier(c)
	if e != nil {
		bm.Error(e)
	}
	b.disableOutput()
	bm.SetParallelism(int(defaultNumberOfConns) / runtime.NumCPU())
	bm.ResetTimer()
	bm.RunParallel(func(pb *testing.PB) {
		done := b.Barrier.done()
		for pb.Next() {
			b.ratelimiter.pace(done)
			b.performSingleRequest()
		}
	})
}
