package lib

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	programName = "bombardier"
)

func TestInvalidArgsParsing(t *testing.T) {
	expectations := []struct {
		in  []string
		out string
	}{
		{
			[]string{programName},
			"required argument 'url' not provided",
		},
		{
			[]string{programName, "http://google.com", "http://yahoo.com"},
			"unexpected http://yahoo.com",
		},
	}
	for _, e := range expectations {
		p := newKingpinParser()
		if _, err := p.Parse(e.in); err == nil ||
			err.Error() != e.out {
			t.Error(err, e.out)
		}
	}
}

func TestUnspecifiedArgParsing(t *testing.T) {
	p := newKingpinParser()
	args := []string{programName, "--someunspecifiedflag"}
	_, err := p.Parse(args)
	if err == nil {
		t.Fail()
	}
}

func TestArgsParsing(t *testing.T) {
	ten := uint64(10)
	expectations := []struct {
		in  [][]string
		out Config
	}{
		{
			[][]string{{programName, "https://somehost.somedomain"}},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"-c", "10",
					"-n", strconv.FormatUint(defaultNumberOfReqs, decBase),
					"-t", "10s",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-c10",
					"-n" + strconv.FormatUint(defaultNumberOfReqs, decBase),
					"-t10s",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--connections", "10",
					"--requests", strconv.FormatUint(defaultNumberOfReqs, decBase),
					"--timeout", "10s",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--connections=10",
					"--requests=" + strconv.FormatUint(defaultNumberOfReqs, decBase),
					"--timeout=10s",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      10,
				Timeout:       10 * time.Second,
				Headers:       new(HeadersList),
				Method:        "GET",
				NumReqs:       &defaultNumberOfReqs,
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--latencies",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-l",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:       defaultNumberOfConns,
				Timeout:        defaultTimeout,
				Headers:        new(HeadersList),
				PrintLatencies: true,
				Method:         "GET",
				Url:            "https://somehost.somedomain",
				PrintIntro:     true,
				PrintProgress:  true,
				PrintResult:    true,
				Format:         knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--insecure",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-k",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Insecure:      true,
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--key", "testclient.key",
					"--cert", "testclient.cert",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--key=testclient.key",
					"--cert=testclient.cert",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				KeyPath:       "testclient.key",
				CertPath:      "testclient.cert",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--method", "POST",
					"--body", "reqbody",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--method=POST",
					"--body=reqbody",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-m", "POST",
					"-b", "reqbody",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-mPOST",
					"-breqbody",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "POST",
				Body:          "reqbody",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--header", "One: Value one",
					"--header", "Two: Value two",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-H", "One: Value one",
					"-H", "Two: Value two",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--header=One: Value one",
					"--header=Two: Value two",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns: defaultNumberOfConns,
				Timeout:  defaultTimeout,
				Headers: &HeadersList{
					{"One", "Value one"},
					{"Two", "Value two"},
				},
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--rate", "10",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-r", "10",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--rate=10",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-r10",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				Rate:          &ten,
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--fasthttp",
					"https://somehost.somedomain",
				},
				{
					programName,
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				ClientType:    fhttp,
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--http1",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				ClientType:    nhttp1,
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--http2",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				ClientType:    nhttp2,
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--body-file=testbody.txt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--body-file", "testbody.txt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-f", "testbody.txt",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				BodyFilePath:  "testbody.txt",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--stream",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-s",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Stream:        true,
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--print=r,i,p",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print", "r,i,p",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-p", "r,i,p",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print=result,i,p",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print", "r,intro,p",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-p", "r,i,progress",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--print=i,r",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print", "i,r",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-p", "i,r",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print=intro,r",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--print", "i,result",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-p", "intro,r",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: false,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--no-print",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-q",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    false,
				PrintProgress: false,
				PrintResult:   false,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--format", "plain-text",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format", "pt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format=plain-text",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format=pt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-o", "plain-text",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-o", "pt",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("plain-text"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--format", "json",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format", "j",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format=json",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format=j",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-o", "json",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-o", "j",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        knownFormat("json"),
			},
		},
		{
			[][]string{
				{
					programName,
					"--format", "path:/path/to/tmpl.txt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"--format=path:/path/to/tmpl.txt",
					"https://somehost.somedomain",
				},
				{
					programName,
					"-o", "path:/path/to/tmpl.txt",
					"https://somehost.somedomain",
				},
			},
			Config{
				NumConns:      defaultNumberOfConns,
				Timeout:       defaultTimeout,
				Headers:       new(HeadersList),
				Method:        "GET",
				Url:           "https://somehost.somedomain",
				PrintIntro:    true,
				PrintProgress: true,
				PrintResult:   true,
				Format:        userDefinedTemplate("/path/to/tmpl.txt"),
			},
		},
	}
	for _, e := range expectations {
		for _, args := range e.in {
			p := newKingpinParser()
			cfg, err := p.Parse(args)
			if err != nil {
				t.Error(err)
				continue
			}
			if !reflect.DeepEqual(cfg, e.out) {
				t.Logf("Expected: %#v", e.out)
				t.Logf("Got: %#v", cfg)
				t.Fail()
			}
		}
	}
}

func TestParsePrintSpec(t *testing.T) {
	exps := []struct {
		spec    string
		results [3]bool
		err     error
	}{
		{
			"",
			[3]bool{},
			errEmptyPrintSpec,
		},
		{
			"a,b,c",
			[3]bool{},
			fmt.Errorf("%q is not a valid part of print spec", "a"),
		},
		{
			"i,p,r,i",
			[3]bool{},
			fmt.Errorf(
				"Spec %q has too many parts, at most 3 are allowed", "i,p,r,i",
			),
		},
		{
			"i",
			[3]bool{true, false, false},
			nil,
		},
		{
			"p",
			[3]bool{false, true, false},
			nil,
		},
		{
			"r",
			[3]bool{false, false, true},
			nil,
		},
		{
			"i,p,r",
			[3]bool{true, true, true},
			nil,
		},
	}
	for _, e := range exps {
		var (
			act = [3]bool{}
			err error
		)
		act[0], act[1], act[2], err = parsePrintSpec(e.spec)
		if !reflect.DeepEqual(err, e.err) {
			t.Errorf("For %q, expected err = %q, but got %q",
				e.spec, e.err, err,
			)
			continue
		}
		if !reflect.DeepEqual(e.results, act) {
			t.Errorf("For %q, expected result = %+v, but got %+v",
				e.spec, e.results, act,
			)
		}
	}
}

func TestArgsParsingWithEmptyPrintSpec(t *testing.T) {
	p := newKingpinParser()
	c, err := p.Parse(
		[]string{programName, "--print=", "somehost.somedomain"})
	if err == nil {
		t.Fail()
	}
	if c != emptyConf {
		t.Fail()
	}
}

func TestArgsParsingWithInvalidPrintSpec(t *testing.T) {
	invalidSpecs := [][]string{
		{programName, "--format", "noprefix.txt", "somehost.somedomain"},
		{programName, "--format=noprefix.txt", "somehost.somedomain"},
		{programName, "-o", "noprefix.txt", "somehost.somedomain"},
		{programName, "--format", "unknown-format", "somehost.somedomain"},
		{programName, "--format=unknown-format", "somehost.somedomain"},
		{programName, "-o", "unknown-format", "somehost.somedomain"},
	}
	p := newKingpinParser()
	for _, is := range invalidSpecs {
		c, err := p.Parse(is)
		if err == nil || c != emptyConf {
			t.Errorf("invalid print spec %q parsed correctly", is)
		}
	}
}
