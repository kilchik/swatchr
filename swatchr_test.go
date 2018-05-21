package main

import "testing"

func TestExtractBtih(t *testing.T) {
	type testCase struct {
		in       string
		expected string
	}
	for _, c := range []testCase{
		testCase{
			"magnet:?xt=urn:btih:44EE80F1C56BADA494667A50856777BA7DDAAE61&amp;tr=http%3A%2F%2Fbt4.t-ru.org%2Fann%3Fmagnet&amp;dn=%D0%A1%D0%BF%D0%B0%D1%81%D1%82%D0%B8%20%D1%80%D1%8F%D0%B4%D0%BE%D0%B2%D0%BE%D0%B3%D0%BE%20%D0%A0%D0%B0%D0%B9%D0%B0%D0%BD%D0%B0%20%2F%20Saving%20Private%20Ryan%20(%D0%A1%D1%82%D0%B8%D0%B2%D0%B5%D0%BD%20%D0%A1%D0%BF%D0%B8%D0%BB%D0%B1%D0%B5%D1%80%D0%B3%20%2F%20Steven%20Spielberg)%20%5B1998%2C%20%D0%A1%D0%A8%D0%90%2C%20%D0%91%D0%BE%D0%B5%D0%B2%D0%B8%D0%BA%2C%20%D0%B4%D1%80%D0%B0%D0%BC%D0%B0%2C%20%D0%B2%D0%BE%D0%B5%D0%BD%D0%BD%D1%8B%D0%B9%2C%20BDRip-AVC%5D%20Dub%20%2B%20Sub",
			"44EE80F1C56BADA494667A50856777BA7DDAAE61",
		},
		testCase{
			"magnet:?xt=urn:btih:44EE80F1C56BADA494667A50856777BA7DDAAE61",
			"44EE80F1C56BADA494667A50856777BA7DDAAE61",
		},
	} {
		out, err := extractBtih(c.in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != c.expected {
			t.Fatalf("got: %s, expected: %s", out, c.expected)
		}
	}
}
