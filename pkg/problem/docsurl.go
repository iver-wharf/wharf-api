package problem

import "net/url"

const docsHost = "iver-wharf.github.io"

// ConvertURLToAbsDocsURL adds schema and sets the host if that has not been set.
func ConvertURLToAbsDocsURL(u url.URL) *url.URL {
	if !u.IsAbs() {
		u.Scheme = "https"
		u.Host = docsHost
	}
	if u.Fragment == "" && u.Host == docsHost {
		u.Fragment = u.Path
		u.Path = "/"
	}
	return &u
}
