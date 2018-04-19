package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
)

func DeflateData(data []byte) ([]byte, error) {
	var bb bytes.Buffer
	gz := gzip.NewWriter(&bb)
	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}
	gz.Close()
	return bb.Bytes(), nil
}

// Write gunzipped data to a Writer
func GunzipWrite(w io.Writer, data []byte) error {
	// Write gzipped data to the client
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

func InflateData(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	return ioutil.ReadAll(gr)
}

