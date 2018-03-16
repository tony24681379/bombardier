package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/cheggaaa/pb"
	fhist "github.com/codesenberg/concurrent/float64/histogram"
	uhist "github.com/codesenberg/concurrent/uint64/histogram"
	"github.com/satori/go.uuid"
	"github.com/tony24681379/bombardier/internal"
)

type Bombardier struct {
	bytesRead, bytesWritten int64

	// HTTP codes
	req1xx uint64
	req2xx uint64
	req3xx uint64
	req4xx uint64
	req5xx uint64
	others uint64

	Conf        Config
	Barrier     completionBarrier
	ratelimiter limiter
	workers     sync.WaitGroup

	timeTaken time.Duration
	latencies *uhist.Histogram
	requests  *fhist.Histogram

	client   client
	doneChan chan struct{}

	// RPS metrics
	rpl   sync.Mutex
	reqs  int64
	start time.Time

	// Errors
	errors *errorMap

	// Progress bar
	bar *pb.ProgressBar

	// Output
	out      io.Writer
	template *template.Template
}

func NewBombardier(c Config) (*Bombardier, error) {
	if err := c.checkArgs(); err != nil {
		return nil, err
	}
	b := new(Bombardier)
	b.Conf = c
	b.latencies = uhist.Default()
	b.requests = fhist.Default()

	if b.Conf.testType() == counted {
		b.bar = pb.New64(int64(*b.Conf.NumReqs))
	} else if b.Conf.testType() == timed {
		b.bar = pb.New64(b.Conf.Duration.Nanoseconds() / 1e9)
		b.bar.ShowCounters = false
		b.bar.ShowPercent = false
	}
	b.bar.ManualUpdate = true

	if b.Conf.testType() == counted {
		b.Barrier = newCountingCompletionBarrier(*b.Conf.NumReqs)
	} else {
		b.Barrier = newTimedCompletionBarrier(*b.Conf.Duration)
	}

	if b.Conf.Rate != nil {
		b.ratelimiter = newBucketLimiter(*b.Conf.Rate)
	} else {
		b.ratelimiter = &nooplimiter{}
	}

	b.out = os.Stdout

	tlsConfig, err := generateTLSConfig(c)
	if err != nil {
		return nil, err
	}

	var (
		pbody *string
		bsp   bodyStreamProducer
	)
	if c.Stream {
		if c.BodyFilePath != "" {
			bsp = func() (io.ReadCloser, error) {
				return os.Open(c.BodyFilePath)
			}
		} else {
			bsp = func() (io.ReadCloser, error) {
				return ioutil.NopCloser(
					proxyReader{strings.NewReader(c.Body)},
				), nil
			}
		}
	} else {
		pbody = &c.Body
		if c.BodyFilePath != "" {
			var bodyBytes []byte
			bodyBytes, err = ioutil.ReadFile(c.BodyFilePath)
			if err != nil {
				return nil, err
			}
			sbody := string(bodyBytes)
			pbody = &sbody
		}
	}

	cc := &clientOpts{
		HTTP2:     false,
		maxConns:  c.NumConns,
		timeout:   c.Timeout,
		tlsConfig: tlsConfig,

		headers:      c.Headers,
		url:          c.Url,
		method:       c.Method,
		body:         pbody,
		bodProd:      bsp,
		bytesRead:    &b.bytesRead,
		bytesWritten: &b.bytesWritten,
	}
	b.client = makeHTTPClient(c.ClientType, cc)

	if !b.Conf.PrintProgress {
		b.bar.Output = ioutil.Discard
		b.bar.NotPrint = true
	}

	b.template, err = b.prepareTemplate()
	if err != nil {
		return nil, err
	}

	b.workers.Add(int(c.NumConns))
	b.errors = newErrorMap()
	b.doneChan = make(chan struct{}, 2)
	return b, nil
}

func makeHTTPClient(clientType clientTyp, cc *clientOpts) client {
	var cl client
	switch clientType {
	case nhttp1:
		cl = newHTTPClient(cc)
	case nhttp2:
		cc.HTTP2 = true
		cl = newHTTPClient(cc)
	case fhttp:
		fallthrough
	default:
		cl = newFastHTTPClient(cc)
	}
	return cl
}

func (b *Bombardier) prepareTemplate() (*template.Template, error) {
	var (
		templateBytes []byte
		err           error
	)
	switch f := b.Conf.Format.(type) {
	case knownFormat:
		templateBytes = f.template()
	case userDefinedTemplate:
		templateBytes, err = ioutil.ReadFile(string(f))
		if err != nil {
			return nil, err
		}
	default:
		panic("format can't be nil at this point, this is a bug")
	}
	outputTemplate, err := template.New("output-template").
		Funcs(template.FuncMap{
			"WithLatencies": func() bool {
				return b.Conf.PrintLatencies
			},
			"FormatBinary": formatBinary,
			"FormatTimeUs": formatTimeUs,
			"FormatTimeUsUint64": func(us uint64) string {
				return formatTimeUs(float64(us))
			},
			"FloatsToArray": func(ps ...float64) []float64 {
				return ps
			},
			"Multiply": func(num, coeff float64) float64 {
				return num * coeff
			},
			"StringToBytes": func(s string) []byte {
				return []byte(s)
			},
			"UUIDV1": uuid.NewV1,
			"UUIDV2": uuid.NewV2,
			"UUIDV3": uuid.NewV3,
			"UUIDV4": uuid.NewV4,
			"UUIDV5": uuid.NewV5,
		}).Parse(string(templateBytes))

	if err != nil {
		return nil, err
	}
	return outputTemplate, nil
}

func (b *Bombardier) writeStatistics(
	code int, msTaken uint64,
) {
	b.latencies.Increment(msTaken)
	b.rpl.Lock()
	b.reqs++
	b.rpl.Unlock()
	var counter *uint64
	switch code / 100 {
	case 1:
		counter = &b.req1xx
	case 2:
		counter = &b.req2xx
	case 3:
		counter = &b.req3xx
	case 4:
		counter = &b.req4xx
	case 5:
		counter = &b.req5xx
	default:
		counter = &b.others
	}
	atomic.AddUint64(counter, 1)
}

func (b *Bombardier) performSingleRequest() {
	code, msTaken, err := b.client.do()
	if err != nil {
		b.errors.add(err)
	}
	b.writeStatistics(code, msTaken)
}

func (b *Bombardier) worker() {
	done := b.Barrier.done()
	for b.Barrier.tryGrabWork() {
		if b.ratelimiter.pace(done) == brk {
			break
		}
		b.performSingleRequest()
		b.Barrier.jobDone()
	}
}

func (b *Bombardier) barUpdater() {
	done := b.Barrier.done()
	for {
		select {
		case <-done:
			b.bar.Set64(b.bar.Total)
			b.bar.Update()
			b.bar.Finish()
			if b.Conf.PrintProgress {
				fmt.Fprintln(b.out, "Done!")
			}
			b.doneChan <- struct{}{}
			return
		default:
			current := int64(b.Barrier.completed() * float64(b.bar.Total))
			b.bar.Set64(current)
			b.bar.Update()
			time.Sleep(b.bar.RefreshRate)
		}
	}
}

func (b *Bombardier) rateMeter() {
	requestsInterval := 10 * time.Millisecond
	if b.Conf.Rate != nil {
		requestsInterval, _ = estimate(*b.Conf.Rate, rateLimitInterval)
	}
	requestsInterval += 10 * time.Millisecond
	ticker := time.NewTicker(requestsInterval)
	defer ticker.Stop()
	tick := ticker.C
	done := b.Barrier.done()
	for {
		select {
		case <-tick:
			b.recordRps()
			continue
		case <-done:
			b.workers.Wait()
			b.recordRps()
			b.doneChan <- struct{}{}
			return
		}
	}
}

func (b *Bombardier) recordRps() {
	b.rpl.Lock()
	duration := time.Since(b.start)
	reqs := b.reqs
	b.reqs = 0
	b.start = time.Now()
	b.rpl.Unlock()

	reqsf := float64(reqs) / duration.Seconds()
	b.requests.Increment(reqsf)
}

func (b *Bombardier) Bombard() {
	if b.Conf.PrintIntro {
		b.printIntro()
	}
	b.bar.Start()
	bombardmentBegin := time.Now()
	b.start = time.Now()
	for i := uint64(0); i < b.Conf.NumConns; i++ {
		go func() {
			defer b.workers.Done()
			b.worker()
		}()
	}
	go b.rateMeter()
	go b.barUpdater()
	b.workers.Wait()
	b.timeTaken = time.Since(bombardmentBegin)
	<-b.doneChan
	<-b.doneChan
}

func (b *Bombardier) printIntro() {
	if b.Conf.testType() == counted {
		fmt.Fprintf(b.out,
			"Bombarding %v with %v request(s) using %v connection(s)\n",
			b.Conf.Url, *b.Conf.NumReqs, b.Conf.NumConns)
	} else if b.Conf.testType() == timed {
		fmt.Fprintf(b.out, "Bombarding %v for %v using %v connection(s)\n",
			b.Conf.Url, *b.Conf.Duration, b.Conf.NumConns)
	}
}

func (b *Bombardier) gatherInfo() internal.TestInfo {
	info := internal.TestInfo{
		Spec: internal.Spec{
			NumberOfConnections: b.Conf.NumConns,

			Method: b.Conf.Method,
			URL:    b.Conf.Url,

			Body:         b.Conf.Body,
			BodyFilePath: b.Conf.BodyFilePath,

			CertPath: b.Conf.CertPath,
			KeyPath:  b.Conf.KeyPath,

			Stream:     b.Conf.Stream,
			Timeout:    b.Conf.Timeout,
			ClientType: internal.ClientType(b.Conf.ClientType),

			Rate: b.Conf.Rate,
		},
		Result: internal.Results{
			BytesRead:    b.bytesRead,
			BytesWritten: b.bytesWritten,
			TimeTaken:    b.timeTaken,

			Req1XX: b.req1xx,
			Req2XX: b.req2xx,
			Req3XX: b.req3xx,
			Req4XX: b.req4xx,
			Req5XX: b.req5xx,
			Others: b.others,

			Latencies: b.latencies,
			Requests:  b.requests,
		},
	}

	testType := b.Conf.testType()
	info.Spec.TestType = internal.TestType(testType)
	if testType == timed {
		info.Spec.TestDuration = *b.Conf.Duration
	} else if testType == counted {
		info.Spec.NumberOfRequests = *b.Conf.NumReqs
	}

	if b.Conf.Headers != nil {
		for _, h := range *b.Conf.Headers {
			info.Spec.Headers = append(info.Spec.Headers,
				internal.Header{
					Key:   h.key,
					Value: h.value,
				})
		}
	}

	for _, ewc := range b.errors.byFrequency() {
		info.Result.Errors = append(info.Result.Errors,
			internal.ErrorWithCount{
				Error: ewc.error,
				Count: ewc.count,
			})
	}

	return info
}

func (b *Bombardier) PrintStats() {
	info := b.gatherInfo()
	err := b.template.Execute(b.out, info)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (b *Bombardier) redirectOutputTo(out io.Writer) {
	b.bar.Output = out
	b.out = out
}

func (b *Bombardier) disableOutput() {
	b.redirectOutputTo(ioutil.Discard)
	b.bar.NotPrint = true
}
