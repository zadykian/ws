package signoz

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetLogsSendsTheRulesBack(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = r.Method + " " + r.URL.Path + " " + string(b)
	}))
	defer srv.Close()
	have := logsTTL{Days: 15, Status: "success", Conditions: []byte(`[{"filter":"x"}]`)}
	c := setLogs{Retention{API: API{URL: srv.URL}, Days: 90}, have}
	if err := c.Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := `POST /api/v2/settings/ttl {"defaultTTLDays":90,"ttlConditions":[{"filter":"x"}],"type":"logs"}`
	if body != want {
		t.Errorf("sent %s\nwant %s", body, want)
	}
}

func TestSetLogsSendsNoRulesAsAList(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}))
	defer srv.Close()
	if err := (setLogs{Retention{API: API{URL: srv.URL}, Days: 90}, logsTTL{}}).Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := `{"defaultTTLDays":90,"ttlConditions":[],"type":"logs"}`; body != want {
		t.Errorf("sent %s, want %s", body, want)
	}
}

func TestLogsDueWhereTheDaysDifferOrTheChangeIsNotDone(t *testing.T) {
	for _, tc := range []struct {
		l    logsTTL
		want bool
	}{
		{logsTTL{Days: 90, Status: "success"}, false},
		{logsTTL{Days: 15, Status: "success"}, true},
		{logsTTL{Days: 90, Status: "failed"}, true},
	} {
		if got := tc.l.due(90); got != tc.want {
			t.Errorf("%v due = %v, want %v", tc.l, got, tc.want)
		}
	}
}
