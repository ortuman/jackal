package xep0198

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeSMID(t *testing.T) {
	smID := "b3J0dW1hbi9Db252ZXJzYXRpb25zLjRSWUUAkWL2Q/wpu3rxkkJYM/CH1g=="

	jd, nonce, err := decodeSMID(smID)

	require.Nil(t, err)
	require.NotNil(t, jd)
	require.NotNil(t, nonce)
}
