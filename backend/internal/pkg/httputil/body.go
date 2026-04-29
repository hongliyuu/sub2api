package httputil

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const (
	requestBodyReadInitCap    = 512
	requestBodyReadMaxInitCap = 1 << 20
)

// ReadRequestBodyWithPrealloc reads request body with preallocated buffer based on content length.
func ReadRequestBodyWithPrealloc(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	bodyReader := req.Body
	contentEncoding := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Encoding")))
	if contentEncoding != "" && contentEncoding != "identity" {
		var err error
		bodyReader, err = decodeRequestBodyReader(req.Body, contentEncoding)
		if err != nil {
			return nil, err
		}
		defer bodyReader.Close()
		req.Header.Del("Content-Encoding")
		req.ContentLength = -1
	}

	capHint := requestBodyReadInitCap
	if req.ContentLength > 0 {
		switch {
		case req.ContentLength < int64(requestBodyReadInitCap):
			capHint = requestBodyReadInitCap
		case req.ContentLength > int64(requestBodyReadMaxInitCap):
			capHint = requestBodyReadMaxInitCap
		default:
			capHint = int(req.ContentLength)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, capHint))
	if _, err := io.Copy(buf, bodyReader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeRequestBodyReader(body io.ReadCloser, contentEncoding string) (io.ReadCloser, error) {
	switch contentEncoding {
	case "gzip":
		return gzip.NewReader(body)
	case "br":
		return io.NopCloser(brotli.NewReader(body)), nil
	case "zstd":
		reader, err := zstd.NewReader(body)
		if err != nil {
			return nil, err
		}
		return &zstdReadCloser{Decoder: reader}, nil
	default:
		return body, nil
	}
}

type zstdReadCloser struct {
	*zstd.Decoder
}

func (r *zstdReadCloser) Close() error {
	r.Decoder.Close()
	return nil
}
