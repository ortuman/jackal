/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package compress

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/stretchr/testify/require"
)

func TestZlibDeflate(t *testing.T) {
	tt := []struct {
		level  config.CompressionLevel
		input  string
		output []byte
	}{
		{
			config.DefaultCompression,
			"My lord, dispatch; read o'er these articles.",
			[]byte{120, 156, 242, 173, 84, 200, 201, 47, 74, 209, 81, 72, 201, 44, 46, 72, 44, 73, 206, 176,
				86, 40, 74, 77, 76, 81, 200, 87, 79, 45, 82, 40, 201, 72, 45, 78, 85, 72, 44, 42, 201, 76, 206, 73, 45,
				214, 3, 0, 0, 0, 255, 255},
		},
		{
			config.BestCompression,
			"Neither, fair saint, if either thee dislike.",
			[]byte{120, 218, 242, 75, 205, 44, 201, 72, 45, 210, 81, 72, 75, 204, 44, 82, 40, 78, 204, 204, 43,
				209, 81, 200, 76, 83, 128, 8, 43, 148, 100, 164, 166, 42, 164, 100, 22, 231, 100, 102, 167, 234, 1, 0,
				0, 0, 255, 255},
		},
		{
			config.SpeedCompression,
			"Call me but love, and I'll be new baptized; Henceforth I never will be Romeo.",
			[]byte{120, 1, 4, 192, 177, 13, 128, 32, 16, 5, 208, 85, 126, 103, 99, 92, 192, 210, 70, 90, 55,
				224, 228, 27, 73, 142, 59, 67, 16, 18, 167, 247, 109, 81, 21, 133, 144, 183, 65, 189, 115, 70, 180, 132,
				48, 169, 66, 8, 227, 128, 196, 167, 229, 143, 105, 197, 78, 59, 121, 121, 109, 55, 2, 140, 157, 21, 35,
				171, 66, 136, 195, 11, 125, 249, 1, 0, 0, 255, 255},
		},
	}
	wBuf := new(bytes.Buffer)
	for _, tc := range tt {
		wBuf.Reset()
		compressor := NewZlibCompressor(nil, wBuf, tc.level)
		compressor.Write([]byte(tc.input))
		require.Equal(t, tc.output, wBuf.Bytes())
	}
}

func TestZlibInflate(t *testing.T) {
	tt := []struct {
		level  config.CompressionLevel
		input  []byte
		output string
	}{
		{
			config.DefaultCompression,
			[]byte{120, 156, 242, 173, 84, 200, 201, 47, 74, 209, 81, 72, 201, 44, 46, 72, 44, 73, 206, 176,
				86, 40, 74, 77, 76, 81, 200, 87, 79, 45, 82, 40, 201, 72, 45, 78, 85, 72, 44, 42, 201, 76, 206, 73, 45,
				214, 3, 0, 0, 0, 255, 255},
			"My lord, dispatch; read o'er these articles.",
		},
		{
			config.BestCompression,
			[]byte{120, 218, 242, 75, 205, 44, 201, 72, 45, 210, 81, 72, 75, 204, 44, 82, 40, 78, 204, 204, 43,
				209, 81, 200, 76, 83, 128, 8, 43, 148, 100, 164, 166, 42, 164, 100, 22, 231, 100, 102, 167, 234, 1, 0,
				0, 0, 255, 255},
			"Neither, fair saint, if either thee dislike.",
		},
		{
			config.SpeedCompression,
			[]byte{120, 1, 4, 192, 177, 13, 128, 32, 16, 5, 208, 85, 126, 103, 99, 92, 192, 210, 70, 90, 55,
				224, 228, 27, 73, 142, 59, 67, 16, 18, 167, 247, 109, 81, 21, 133, 144, 183, 65, 189, 115, 70, 180, 132,
				48, 169, 66, 8, 227, 128, 196, 167, 229, 143, 105, 197, 78, 59, 121, 121, 109, 55, 2, 140, 157, 21, 35,
				171, 66, 136, 195, 11, 125, 249, 1, 0, 0, 255, 255},
			"Call me but love, and I'll be new baptized; Henceforth I never will be Romeo.",
		},
	}
	rBuf := new(bytes.Buffer)
	for _, tc := range tt {
		rBuf.Reset()
		rBuf.Write(tc.input)
		compressor := NewZlibCompressor(rBuf, nil, tc.level)
		b, _ := ioutil.ReadAll(compressor)
		require.Equal(t, tc.output, string(b))
	}
}

func TestInvalidCompressionLevel(t *testing.T) {
	compressor := NewZlibCompressor(new(bytes.Buffer), new(bytes.Buffer), config.CompressionLevel(100))
	_, err := compressor.Write([]byte("Failing!"))
	require.NotNil(t, err)
}

func TestInvalidInflate(t *testing.T) {
	rBuf := new(bytes.Buffer)
	rBuf.Write([]byte("this is garbage!"))
	compressor := NewZlibCompressor(rBuf, nil, config.DefaultCompression)
	_, err := ioutil.ReadAll(compressor)
	require.NotNil(t, err)
}
