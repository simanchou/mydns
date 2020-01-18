package lkvs

import "testing"

func TestFindSubDomain(t *testing.T) {
	var subDomainTests = []struct {
		in       string
		expected string
	}{
		{"a.siman.com", "a"},
		{"b.siman.com", "b"},
		{"c.siman.com", "c"},
	}

	for _, tt := range subDomainTests {
		actual := FindSubDomain(tt.in, "siman.com")
		if actual != tt.expected {
			t.Errorf("FindSubDomain(%s) = %s; expectd %s\n",
				tt.in, actual, tt.expected)
		}
	}
}
