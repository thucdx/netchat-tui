package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GetTeamsForUser tests
// ---------------------------------------------------------------------------

func TestGetTeamsForUser_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `[
		{"id":"team-1","name":"alpha","display_name":"Alpha Team"},
		{"id":"team-2","name":"beta","display_name":"Beta Team"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	teams, err := c.GetTeamsForUser("user-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}

	// Verify first team fields.
	if teams[0].ID != "team-1" {
		t.Errorf("teams[0].ID: got %q, want %q", teams[0].ID, "team-1")
	}
	if teams[0].Name != "alpha" {
		t.Errorf("teams[0].Name: got %q, want %q", teams[0].Name, "alpha")
	}
	if teams[0].DisplayName != "Alpha Team" {
		t.Errorf("teams[0].DisplayName: got %q, want %q", teams[0].DisplayName, "Alpha Team")
	}

	// Verify second team fields.
	if teams[1].ID != "team-2" {
		t.Errorf("teams[1].ID: got %q, want %q", teams[1].ID, "team-2")
	}
	if teams[1].Name != "beta" {
		t.Errorf("teams[1].Name: got %q, want %q", teams[1].Name, "beta")
	}
	if teams[1].DisplayName != "Beta Team" {
		t.Errorf("teams[1].DisplayName: got %q, want %q", teams[1].DisplayName, "Beta Team")
	}
}

func TestGetTeamsForUser_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	teams, err := c.GetTeamsForUser("user-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(teams) != 0 {
		t.Errorf("expected empty slice, got %d teams", len(teams))
	}
}

func TestGetTeamsForUser_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetTeamsForUser("user-abc")
	if err == nil {
		t.Fatal("expected an error for 500 response, got nil")
	}
}

func TestGetTeamsForUser_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json at all {`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetTeamsForUser("user-abc")
	if err == nil {
		t.Fatal("expected an error for invalid JSON response, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetFirstTeam tests
// ---------------------------------------------------------------------------

func TestGetFirstTeam_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `[
		{"id":"team-1","name":"first","display_name":"First Team"},
		{"id":"team-2","name":"second","display_name":"Second Team"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	team, err := c.GetFirstTeam("user-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if team.ID != "team-1" {
		t.Errorf("expected first team ID %q, got %q", "team-1", team.ID)
	}
	if team.Name != "first" {
		t.Errorf("expected first team Name %q, got %q", "first", team.Name)
	}
	if team.DisplayName != "First Team" {
		t.Errorf("expected first team DisplayName %q, got %q", "First Team", team.DisplayName)
	}
}

func TestGetFirstTeam_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetFirstTeam("user-abc")
	if err == nil {
		t.Fatal("expected an error when no teams found, got nil")
	}

	if !strings.Contains(err.Error(), "no teams found") {
		t.Errorf("expected error to contain %q, got: %q", "no teams found", err.Error())
	}
}
