// Package response contains plain old Go types returned by wharf-web in the
// HTTP responses, with Swaggo-specific Go tags.
package response

// Token holds credentials for a remote provider.
type Token struct {
	TokenID  uint   `json:"tokenId"`
	Token    string `json:"token" format:"password"`
	UserName string `json:"userName"`
}
