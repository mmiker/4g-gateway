package ini

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var (
	_ dataSource = (*sourceFile)(nil)
	_ dataSource = (*sourceData)(nil)
	_ dataSource = (*sourceReadCloser)(nil)
)

// dataSource is an interface that returns object which can be read and closed.
type dataSource interface {
	ReadCloser() (io.ReadCloser, error)
}

// sourceFile represents an object that contains content on the local file system.
type sourceFile struct {
	name string
}

func (s sourceFile) ReadCloser() (_ io.ReadCloser, err error) {
	return os.Open(s.name)
}

// sourceData represents an object that contains content in memory.
type sourceData struct {
	data []byte
}

func (s *sourceData) ReadCloser() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(s.data)), nil
}

// sourceReadCloser represents an input stream with Close method.
type sourceReadCloser struct {
	reader io.ReadCloser
}

func (s *sourceReadCloser) ReadCloser() (io.ReadCloser, error) {
	return s.reader, nil
}

func parseDataSource(source interface{}) (dataSource, error) {
	switch s := source.(type) {
	case string:
		return sourceFile{s}, nil
	case []byte:
		return &sourceData{s}, nil
	case io.ReadCloser:
		return &sourceReadCloser{s}, nil
	case io.Reader:
		return &sourceReadCloser{ioutil.NopCloser(s)}, nil
	default:
		return nil, fmt.Errorf("error parsing data source: unknown type %q", s)
	}
}
