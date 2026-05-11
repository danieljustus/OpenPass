package serverbootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()
	//nolint:errchkjson
	_ = json.NewEncoder(buf).Encode(data)
	_, _ = w.Write(buf.Bytes())
}

func handleOAuthProtectedResource(bind string, port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := fmt.Sprintf("http://%s:%d", bind, port)
		writeJSON(w, http.StatusOK, map[string]any{
			"resource":                 addr + "/mcp",
			"bearer_methods_supported": []string{"header"},
			"resource_name":            "OpenPass MCP Server",
		})
	}
}

func handleOAuthAuthorizationServer(bind string, port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := fmt.Sprintf("http://%s:%d", bind, port)
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer":                                addr,
			"authorization_endpoint":                addr + "/mcp/oauth/authorize",
			"token_endpoint":                        addr + "/mcp/oauth/token",
			"response_types_supported":              []string{"code"},
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		})
	}
}

func handleOAuthAuthorizeStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error":             "not_implemented",
		"error_description": "OAuth authorization is not yet implemented. Use bearer token authentication.",
	})
}

func handleOAuthTokenStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error":             "not_implemented",
		"error_description": "OAuth token exchange is not yet implemented. Use bearer token authentication.",
	})
}
