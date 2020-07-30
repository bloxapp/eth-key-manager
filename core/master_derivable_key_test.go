package core

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		testName      string
		seed          []byte
		expectedError string
	}{
		{
			testName:      "seed length 0",
			seed:          _byteArray(""),
			expectedError: "seed can't be nil or of length different than 32",
		},
		{
			testName:      "seed length 29",
			seed:          _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1"),
			expectedError: "seed can't be nil or of length different than 32",
		},
		{
			testName:      "seed nil",
			seed:          nil,
			expectedError: "seed can't be nil or of length different than 32",
		},
		{
			testName:      "seed nil",
			seed:          _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fff"),
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			_, err := MasterKeyFromSeed(test.seed)
			if len(test.expectedError) != 0 {
				require.EqualError(t, err, test.expectedError)
				return
			}
			require.Nil(t, err)
		})
	}
}