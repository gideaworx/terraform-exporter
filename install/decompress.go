package install

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"io"
)

func (i *Command) getDecompressedReader(reader io.Reader) (io.ReadCloser, error) {
	br := bufio.NewReader(reader)
	for magic, decompressor := range i.decompressors {
		if headerBytes, err := br.Peek(len(magic)); err == nil {
			if string(headerBytes) == magic {
				return decompressor(reader)
			}
		}
	}

	return i.nopDecompressor(reader)
}

func (i *Command) gzipDecompressor(reader io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(reader)
}

func (i *Command) bzip2Decompressor(reader io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(bzip2.NewReader(reader)), nil
}

func (i *Command) nopDecompressor(reader io.Reader) (io.ReadCloser, error) {
	if rc, ok := reader.(io.ReadCloser); ok {
		return rc, nil
	}

	return io.NopCloser(reader), nil
}
