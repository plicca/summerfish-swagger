package assetfs

import "testing"

func TestAddPrefix(t *testing.T) {
	tp := []struct {
		Prefix  string
		Pattern string
		Want    string
	}{
		{"", "", "/"},
		{"", "/", "/"},
		{"/", "/", "/"},
		{"///", "///", "/"},
		{"/", "a", "/a"},
		{"/", "/a", "/a"},
		{"/", "a/", "/a/"},
		{"/", "/a/", "/a/"},
	}
	for n, el := range tp {
		uri := AddPrefix(el.Prefix, el.Pattern)
		if uri != el.Want {
			t.Fatalf("Item %d %#v failed: Want %q, have %q",
				n, el, el.Want, uri)
		}
	}
}
