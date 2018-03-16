package lib

import (
	"testing"
	"time"
)

var (
	defaultNumberOfReqs = uint64(10000)
)

func TestCanHaveBody(t *testing.T) {
	expectations := []struct {
		in  string
		out bool
	}{
		{"GET", false},
		{"POST", true},
		{"PUT", true},
		{"DELETE", true},
		{"HEAD", false},
		{"OPTIONS", true},
	}
	for _, e := range expectations {
		if r := canHaveBody(e.in); r != e.out {
			t.Error(e.in, e.out, r)
		}
	}
}

func TestAllowedHttpMethod(t *testing.T) {
	expectations := []struct {
		in  string
		out bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", true},
		{"DELETE", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"TRUNCATE", false},
	}
	for _, e := range expectations {
		if r := allowedHTTPMethod(e.in); r != e.out {
			t.Logf("Expected f(%v) = %v, but got %v", e.in, e.out, r)
			t.Fail()
		}
	}
}

func TestCheckArgs(t *testing.T) {
	invalidNumberOfReqs := uint64(0)
	smallTestDuration := 99 * time.Millisecond
	negativeTimeoutDuration := -1 * time.Second
	noHeaders := new(HeadersList)
	zeroRate := uint64(0)
	expectations := []struct {
		in  Config
		out error
	}{
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "ftp://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			errInvalidURL,
		},
		{
			Config{
				NumConns: 0,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			errInvalidNumberOfConns,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &invalidNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			errInvalidNumberOfRequests,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  nil,
				Duration: &smallTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			errInvalidTestDuration,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  negativeTimeoutDuration,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			errNegativeTimeout,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "BODY",
				Format:   knownFormat("plain-text"),
			},
			errBodyNotAllowed,
		},
		{
			Config{
				NumConns:     defaultNumberOfConns,
				NumReqs:      &defaultNumberOfReqs,
				Duration:     &defaultTestDuration,
				Url:          "http://localhost:8080",
				Headers:      noHeaders,
				Timeout:      defaultTimeout,
				Method:       "GET",
				BodyFilePath: "testbody.txt",
				Format:       knownFormat("plain-text"),
			},
			errBodyNotAllowed,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				Format:   knownFormat("plain-text"),
			},
			nil,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				CertPath: "test_cert.pem",
				KeyPath:  "",
				Format:   knownFormat("plain-text"),
			},
			errNoPathToKey,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Body:     "",
				CertPath: "",
				KeyPath:  "test_key.pem",
				Format:   knownFormat("plain-text"),
			},
			errNoPathToCert,
		},
		{
			Config{
				NumConns: defaultNumberOfConns,
				NumReqs:  &defaultNumberOfReqs,
				Duration: &defaultTestDuration,
				Url:      "http://localhost:8080",
				Headers:  noHeaders,
				Timeout:  defaultTimeout,
				Method:   "GET",
				Rate:     &zeroRate,
				Format:   knownFormat("plain-text"),
			},
			errZeroRate,
		},
		{
			Config{
				NumConns:     defaultNumberOfConns,
				NumReqs:      &defaultNumberOfReqs,
				Duration:     &defaultTestDuration,
				Url:          "http://localhost:8080",
				Headers:      noHeaders,
				Timeout:      defaultTimeout,
				Method:       "POST",
				Body:         "abracadabra",
				BodyFilePath: "testbody.txt",
				Format:       knownFormat("plain-text"),
			},
			errBodyProvidedTwice,
		},
	}
	for _, e := range expectations {
		if r := e.in.checkArgs(); r != e.out {
			t.Logf("Expected (%v).checkArgs to return %v, but got %v", e.in, e.out, r)
			t.Fail()
		}
		if _, r := NewBombardier(e.in); r != e.out {
			t.Logf("Expected newBombardier(%v) to return %v, but got %v", e.in, e.out, r)
			t.Fail()
		}
	}
}

func TestCheckArgsGarbageUrl(t *testing.T) {
	c := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  &defaultNumberOfReqs,
		Duration: &defaultTestDuration,
		Url:      "8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	if c.checkArgs() == nil {
		t.Fail()
	}
}

func TestCheckArgsInvalidRequestMethod(t *testing.T) {
	c := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  &defaultNumberOfReqs,
		Duration: &defaultTestDuration,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "ABRACADABRA",
		Body:     "",
	}
	e := c.checkArgs()
	if e == nil {
		t.Fail()
	}
	if _, ok := e.(*invalidHTTPMethodError); !ok {
		t.Fail()
	}
}

func TestCheckArgsTestType(t *testing.T) {
	countedConfig := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  &defaultNumberOfReqs,
		Duration: nil,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	timedConfig := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  nil,
		Duration: &defaultTestDuration,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	both := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  &defaultNumberOfReqs,
		Duration: &defaultTestDuration,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	defaultConfig := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  nil,
		Duration: nil,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	if err := countedConfig.checkArgs(); err != nil ||
		countedConfig.testType() != counted {
		t.Fail()
	}
	if err := timedConfig.checkArgs(); err != nil ||
		timedConfig.testType() != timed {
		t.Fail()
	}
	if err := both.checkArgs(); err != nil ||
		both.testType() != counted {
		t.Fail()
	}
	if err := defaultConfig.checkArgs(); err != nil ||
		defaultConfig.testType() != timed ||
		defaultConfig.Duration != &defaultTestDuration {
		t.Fail()
	}
}

func TestTimeoutMillis(t *testing.T) {
	defaultConfig := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  nil,
		Duration: nil,
		Url:      "http://localhost:8080",
		Headers:  nil,
		Timeout:  2 * time.Second,
		Method:   "GET",
		Body:     "",
	}
	if defaultConfig.timeoutMillis() != 2000000 {
		t.Fail()
	}
}

func TestInvalidHTTPMethodError(t *testing.T) {
	invalidMethod := "NOSUCHMETHOD"
	want := "Unknown HTTP method: " + invalidMethod
	err := &invalidHTTPMethodError{invalidMethod}
	if got := err.Error(); got != want {
		t.Error(got, want)
	}
}

func TestParsingOfURLsWithoutScheme(t *testing.T) {
	c := Config{
		NumConns: defaultNumberOfConns,
		NumReqs:  nil,
		Duration: nil,
		Url:      "localhost:8080",
		Headers:  new(HeadersList),
		Timeout:  defaultTimeout,
		Method:   "GET",
		Body:     "",
	}
	if err := c.checkArgs(); err != nil {
		t.Error(err)
		return
	}
	exp := "http://localhost:8080"
	if act := c.Url; act != exp {
		t.Error(exp, act)
	}
}

func TestClientTypToStringConversion(t *testing.T) {
	expectations := []struct {
		in  clientTyp
		out string
	}{
		{fhttp, "FastHTTP"},
		{nhttp1, "net/http v1.x"},
		{nhttp2, "net/http v2.0"},
		{42, "unknown client"},
	}
	for _, exp := range expectations {
		act := exp.in.String()
		if act != exp.out {
			t.Errorf("Expected %v, but got %v", exp.out, act)
		}
	}
}

func clientTypeFromString(s string) clientTyp {
	switch s {
	case "fasthttp":
		return fhttp
	case "http1":
		return nhttp1
	case "http2":
		return nhttp2
	default:
		return fhttp
	}
}
