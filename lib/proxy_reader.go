package lib

import "io"

type proxyReader struct {
	io.Reader
}
