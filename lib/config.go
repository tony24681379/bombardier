package lib

import (
	"fmt"
	"sort"
	"time"

	"github.com/goware/urlx"
)

type Config struct {
	NumConns                       uint64
	NumReqs                        *uint64
	Duration                       *time.Duration
	Url, Method, CertPath, KeyPath string
	Body, BodyFilePath             string
	Stream                         bool
	Headers                        *HeadersList
	Timeout                        time.Duration
	// TODO(codesenberg): PrintLatencies should probably be
	// re(named&maked) into printPercentiles or even let
	// users provide their own percentiles and not just
	// calculate for [0.5, 0.75, 0.9, 0.99]
	PrintLatencies, Insecure bool
	Rate                     *uint64
	ClientType               clientTyp

	PrintIntro, PrintProgress, PrintResult bool

	Format format
}

type testTyp int

const (
	none testTyp = iota
	timed
	counted
)

type invalidHTTPMethodError struct {
	method string
}

func (i *invalidHTTPMethodError) Error() string {
	return fmt.Sprintf("Unknown HTTP method: %v", i.method)
}

func (c *Config) checkArgs() error {
	c.checkOrSetDefaultTestType()

	checks := []func() error{
		c.checkURL,
		c.checkRate,
		c.checkRunParameters,
		c.checkTimeoutDuration,
		c.checkHTTPParameters,
		c.checkCertPaths,
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) checkOrSetDefaultTestType() {
	if c.testType() == none {
		c.Duration = &defaultTestDuration
	}
}

func (c *Config) testType() testTyp {
	typ := none
	if c.NumReqs != nil {
		typ = counted
	} else if c.Duration != nil {
		typ = timed
	}
	return typ
}

func (c *Config) checkURL() error {
	url, err := urlx.Parse(c.Url)
	if err != nil {
		return err
	}
	if url.Host == "" || (url.Scheme != "http" && url.Scheme != "https") {
		return errInvalidURL
	}
	c.Url = url.String()
	return nil
}

func (c *Config) checkRate() error {
	if c.Rate != nil && *c.Rate < 1 {
		return errZeroRate
	}
	return nil
}

func (c *Config) checkRunParameters() error {
	if c.NumConns < uint64(1) {
		return errInvalidNumberOfConns
	}
	if c.testType() == counted && *c.NumReqs < uint64(1) {
		return errInvalidNumberOfRequests
	}
	if c.testType() == timed && *c.Duration < time.Second {
		return errInvalidTestDuration
	}
	return nil
}

func (c *Config) checkTimeoutDuration() error {
	if c.Timeout < 0 {
		return errNegativeTimeout
	}
	return nil
}

func (c *Config) checkHTTPParameters() error {
	if !allowedHTTPMethod(c.Method) {
		return &invalidHTTPMethodError{method: c.Method}
	}
	if !canHaveBody(c.Method) && (c.Body != "" || c.BodyFilePath != "") {
		return errBodyNotAllowed
	}
	if c.Body != "" && c.BodyFilePath != "" {
		return errBodyProvidedTwice
	}
	return nil
}

func (c *Config) checkCertPaths() error {
	if c.CertPath != "" && c.KeyPath == "" {
		return errNoPathToKey
	} else if c.CertPath == "" && c.KeyPath != "" {
		return errNoPathToCert
	}
	return nil
}

func (c *Config) timeoutMillis() uint64 {
	return uint64(c.Timeout.Nanoseconds() / 1000)
}

func allowedHTTPMethod(method string) bool {
	i := sort.SearchStrings(httpMethods, method)
	return i < len(httpMethods) && httpMethods[i] == method
}

func canHaveBody(method string) bool {
	i := sort.SearchStrings(cantHaveBody, method)
	return !(i < len(cantHaveBody) && cantHaveBody[i] == method)
}

type clientTyp int

const (
	fhttp clientTyp = iota
	nhttp1
	nhttp2
)

func (ct clientTyp) String() string {
	switch ct {
	case fhttp:
		return "FastHTTP"
	case nhttp1:
		return "net/http v1.x"
	case nhttp2:
		return "net/http v2.0"
	}
	return "unknown client"
}
