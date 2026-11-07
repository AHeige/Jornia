package foreverdb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOriginIdentityPaginationAndNoCredentialRedirect(t *testing.T) {
	for _, path := range []string{"https://evil.test/api/v1/items", "//evil.test/api/v1/items", "/api/v1/items/1?token=bad", "/api/v1/../admin", "/api/v1/items/1#secret"} {
		if AllowedPath(path) {
			t.Fatal(path)
		}
	}
	for _, path := range []string{"/api/v1/meta", "/api/v1/quests?limit=100&offset=100", "/api/v1/classes/mage"} {
		if !AllowedPath(path) {
			t.Fatal(path)
		}
	}
	mode := "valid"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-API-Key") != "synthetic-key" || !strings.HasPrefix(r.UserAgent(), "Wisp/") {
			t.Error("missing declared identity")
		}
		switch mode {
		case "redirect":
			http.Redirect(w, r, "https://evil.test", 302)
		case "wrong-client":
			fmt.Fprint(w, `{"data":[],"meta":{"build":"1.60.1.70790","client":"other"}}`)
		case "next":
			fmt.Fprint(w, `{"data":[],"meta":{"build":"1.60.1.70790","client":"dev-test","next":"https://evil.test"}}`)
		default:
			fmt.Fprint(w, `{"data":[],"meta":{"build":"1.60.1.70790","client":"dev-test"}}`)
		}
	}))
	defer server.Close()
	c := Client{Key: "synthetic-key", DeveloperID: "dev-test", BaseURL: server.URL, Interval: time.Nanosecond}
	if _, err := c.Fetch(context.Background(), "/api/v1/meta"); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"wrong-client", "next", "redirect"} {
		mode = v
		if _, err := c.Fetch(context.Background(), "/api/v1/meta"); err == nil {
			t.Fatal(v)
		}
	}
	if calls != 4 {
		t.Fatal("redirect followed or retried", calls)
	}
}
