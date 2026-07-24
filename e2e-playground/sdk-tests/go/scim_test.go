package sdktests

// SCIM — a deliberately hybrid test, and the split is the architecturally
// correct one, not a workaround:
//
//   - The ADMIN/config plane around SCIM IS wrapped by the SDK and is driven
//     through it: CreateOrganization, CreateScimEndpoint (+ its one-time bearer
//     token), GetScimEndpoint, RotateScimToken, DeleteScimEndpoint, and
//     AddWebhook for the provisioning lifecycle events. These are the calls a
//     back-office integrator makes, so they belong in the SDK and are tested
//     there.
//
//   - The SCIM /scim/v2/Users CRUD itself is a standardized RFC 7644 REST
//     protocol that an external IdP (Okta/Azure AD) hits directly with a bearer
//     token — it is intentionally NOT wrapped by authorizer-go (an "SDK" for it
//     would just be a generic SCIM HTTP client). Those calls stay raw HTTP and
//     are labelled as the SCIM protocol surface, not an SDK gap.
//
// The webhook-delivery assertion ties both halves together: a webhook
// registered via the SDK admin client must fire (with a valid HMAC signature)
// when a user is provisioned/updated/deprovisioned via the raw SCIM protocol.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"testing"
	"time"

	authorizer "github.com/authorizerdev/authorizer-go/v2"
	authorizerv1 "github.com/authorizerdev/authorizer-proto-go/authorizer/v1"
)

// scimReq issues a raw SCIM protocol request with the endpoint's bearer token.
func scimReq(t *testing.T, method, token, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("scim new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("Origin", baseURL)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("scim %s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return res.StatusCode, parsed
}

// TestSCIM_AdminLifecycleAndProvisioning drives the SDK admin surface for a SCIM
// endpoint, then provisions/updates/deprovisions a user over raw SCIM.
func TestSCIM_AdminLifecycleAndProvisioning(t *testing.T) {
	admin := adminClient(t, baseURL)
	org := newOrg(t, admin, "scim")

	// --- SDK admin: provision the SCIM endpoint + read it back ---
	created, err := admin.CreateScimEndpoint(&authorizer.CreateScimEndpointRequest{OrgID: org.ID})
	if err != nil {
		t.Fatalf("CreateScimEndpoint (SDK): %v", err)
	}
	if created.Token == "" || created.ScimEndpoint == nil {
		t.Fatalf("CreateScimEndpoint returned empty token/endpoint: %+v", created)
	}
	token := created.Token

	ep, err := admin.GetScimEndpoint(&authorizer.ScimEndpointRequest{OrgID: org.ID})
	if err != nil {
		t.Fatalf("GetScimEndpoint (SDK): %v", err)
	}
	if ep.OrgID != org.ID || !ep.Enabled {
		t.Fatalf("unexpected SCIM endpoint: %+v", ep)
	}

	// --- raw SCIM protocol: create → patch(active:false) → delete ---
	email := "scim-user-" + org.ID + "@example.com"
	status, body := scimReq(t, http.MethodPost, token, "/scim/v2/Users", map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": email,
		"name":     map[string]any{"givenName": "Katherine", "familyName": "Johnson"},
		"emails":   []map[string]any{{"value": email, "primary": true}},
		"active":   true,
	})
	if status != http.StatusCreated {
		t.Fatalf("SCIM create: expected 201, got %d (%v)", status, body)
	}
	userID, _ := body["id"].(string)
	if userID == "" || body["userName"] != email {
		t.Fatalf("SCIM create returned unexpected body: %v", body)
	}

	status, _ = scimReq(t, http.MethodPatch, token, "/scim/v2/Users/"+userID, map[string]any{
		"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{{"op": "replace", "value": map[string]any{"active": false}}},
	})
	if status != http.StatusOK {
		t.Fatalf("SCIM patch: expected 200, got %d", status)
	}

	status, _ = scimReq(t, http.MethodDelete, token, "/scim/v2/Users/"+userID, nil)
	if status != http.StatusNoContent {
		t.Fatalf("SCIM delete: expected 204, got %d", status)
	}

	// --- SDK admin: rotate the token (old one must stop working) ---
	rotated, err := admin.RotateScimToken(&authorizer.ScimEndpointRequest{OrgID: org.ID})
	if err != nil {
		t.Fatalf("RotateScimToken (SDK): %v", err)
	}
	if rotated.Token == "" || rotated.Token == token {
		t.Fatalf("RotateScimToken did not return a fresh token")
	}
	if status, _ := scimReq(t, http.MethodGet, token, "/scim/v2/Users", nil); status == http.StatusOK {
		t.Fatalf("old SCIM token still valid after rotation (status %d)", status)
	}

	// --- SDK admin: delete the endpoint (destructive) ---
	if _, err := admin.DeleteScimEndpoint(&authorizer.ScimEndpointRequest{OrgID: org.ID}); err != nil {
		t.Fatalf("DeleteScimEndpoint (SDK): %v", err)
	}
}

// TestSCIM_ProvisioningWebhookDelivery registers the three SCIM lifecycle
// webhooks via the SDK admin client, then verifies real delivery (with valid
// HMAC) to webhook-sink when a user is provisioned/updated/deprovisioned over
// raw SCIM. Runs against the default `authorizer` instance because that is the
// only one configured with --env=e2e (so the docker-private webhook-sink is
// reachable).
func TestSCIM_ProvisioningWebhookDelivery(t *testing.T) {
	admin := adminClient(t, baseURL)
	endpoint := webhookSinkURL + "/webhook"

	for _, event := range []string{"user.provisioned", "user.scim_updated", "user.deprovisioned"} {
		if _, err := admin.AddWebhook(&authorizerv1.AddWebhookRequest{
			EventName: event,
			Endpoint:  endpoint,
			Enabled:   true,
		}); err != nil {
			t.Fatalf("AddWebhook(%s) (SDK): %v", event, err)
		}
	}

	org := newOrg(t, admin, "scim-webhook")
	created, err := admin.CreateScimEndpoint(&authorizer.CreateScimEndpointRequest{OrgID: org.ID})
	if err != nil {
		t.Fatalf("CreateScimEndpoint: %v", err)
	}
	token := created.Token
	email := "scim-webhook-user-" + org.ID + "@example.com"

	status, body := scimReq(t, http.MethodPost, token, "/scim/v2/Users", map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": email,
		"name":     map[string]any{"givenName": "Katherine", "familyName": "Johnson"},
		"emails":   []map[string]any{{"value": email, "primary": true}},
		"active":   true,
	})
	if status != http.StatusCreated {
		t.Fatalf("SCIM create: expected 201, got %d", status)
	}
	userID, _ := body["id"].(string)

	status, _ = scimReq(t, http.MethodPatch, token, "/scim/v2/Users/"+userID, map[string]any{
		"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{{"op": "replace", "path": "name.givenName", "value": "Kate"}},
	})
	if status != http.StatusOK {
		t.Fatalf("SCIM patch: expected 200, got %d", status)
	}

	status, _ = scimReq(t, http.MethodDelete, token, "/scim/v2/Users/"+userID, nil)
	if status != http.StatusNoContent {
		t.Fatalf("SCIM delete: expected 204, got %d", status)
	}

	// Delivery is a detached goroutine; poll the sink until all three land.
	want := []string{"user.deprovisioned", "user.provisioned", "user.scim_updated"}
	var events map[string]webhookDelivery
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		events = fetchWebhookEvents(t, email)
		got := make([]string, 0, len(events))
		for k := range events {
			got = append(got, k)
		}
		sort.Strings(got)
		if equalStrings(got, want) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	for _, event := range want {
		d, ok := events[event]
		if !ok {
			t.Fatalf("missing webhook delivery for %s (got %v)", event, keys(events))
		}
		if d.Body.EventName != event {
			t.Errorf("delivery %s carried event_name %q", event, d.Body.EventName)
		}
		if d.Body.User.Email != email {
			t.Errorf("delivery %s carried user email %q, want %q", event, d.Body.User.Email, email)
		}
		mac := hmac.New(sha256.New, []byte(clientSecret))
		mac.Write([]byte(d.RawBody))
		if want := hex.EncodeToString(mac.Sum(nil)); want != d.Signature {
			t.Errorf("HMAC mismatch for %s: got %s want %s", event, d.Signature, want)
		}
	}
}

type webhookDelivery struct {
	Signature string `json:"signature"`
	RawBody   string `json:"rawBody"`
	Body      struct {
		EventName string `json:"event_name"`
		User      struct {
			Email string `json:"email"`
		} `json:"user"`
	} `json:"body"`
}

func fetchWebhookEvents(t *testing.T, email string) map[string]webhookDelivery {
	t.Helper()
	res, err := http.Get(webhookSinkURL + "/webhook/" + url.PathEscape(email))
	if err != nil || res.StatusCode != http.StatusOK {
		if res != nil {
			res.Body.Close()
		}
		return nil
	}
	defer res.Body.Close()
	var wrap struct {
		Events map[string]webhookDelivery `json:"events"`
	}
	_ = json.NewDecoder(res.Body).Decode(&wrap)
	return wrap.Events
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func keys(m map[string]webhookDelivery) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
