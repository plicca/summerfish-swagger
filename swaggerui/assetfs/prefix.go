package assetfs

import "path"

// AddPrefix joins prefix and pattern and returns a sanitized pattern.
// e.g. prefix=/a and pattern=///b/ results in path=/a/b/
func AddPrefix(prefix, pattern string) string {
	if len(prefix) == 0 || prefix[0] != '/' {
		prefix = "/"
	}
	if len(pattern) == 0 {
		return prefix
	}
	if prefix != pattern && pattern[len(pattern)-1:][0] == '/' {
		return path.Join(prefix, pattern) + "/"
	}
	return path.Join(prefix, pattern)
}
