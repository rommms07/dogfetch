package utils

import (
	"testing"
)

type TestCase struct {
	input    string
	expected string
}

func Test_getSha512Sum(t *testing.T) {
	tests := []*TestCase{
		{
			input:    "Hello, world",
			expected: "f986313ffca1a20c61fa2cff5cb597f1af10a650aecca497a746e8d11d1b6bf33e9e6a25eb7ba26af2fcfaa70472d8250b908419a188a16e17191fc26f423f52",
		},
		{
			input:    "Go is awesome",
			expected: "aed019d53394f4ab14bc75e25332e86966bc7f755694173a9978315e91ec7fe17fa52399f31b530b5f1e2f51902058cccfb8a3e28ab5c2b4bf7371bc1458c963",
		},
		{
			input:    "Roughly speaking, the more one pays for food, the more sweat and spittle one is obliged to eat with it.",
			expected: "e931807d4e01d3b3fd520d9b5c766070e8e3d9fb06c761a2c90f0f4d979ac5cff6e643af300e7176f7cc84c83a4a9c208f14da5bf23ea7bc1c7a2e4c7aa0a665",
		},
	}

	for _, T := range tests {
		res := getSha512Sum(T.input)

		if res != T.expected {
			t.Errorf("(fail) input: %s | out: %s", T.input, res)
		}
	}
}
