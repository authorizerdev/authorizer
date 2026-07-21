package scim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGroupPatch exercises the SCIM Group PATCH member parser against the
// RFC 7644 §3.5.2 shapes AND the two confirmed real-world Entra deviations, plus
// Okta's filtered-path remove. These are the load-bearing interop fixtures: Okta
// and Entra only ever write membership via PATCH on the group's `members`.
func TestParseGroupPatch(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantDisplay     *string
		wantExternalID  *string
		wantOps         []MemberOpJSON
		wantInvalidPath bool
		wantParsedOK    bool
	}{
		{
			name:         "RFC/Okta add — path members, value [{value}]",
			body:         `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"add","path":"members","value":[{"value":"u1"},{"value":"u2"}]}]}`,
			wantOps:      []MemberOpJSON{{Op: "add", Members: []string{"u1", "u2"}}},
			wantParsedOK: true,
		},
		{
			name: "Entra add — capitalized Op",
			// Entra sends "Add"/"Replace"/"Remove". RFC says op is case-insensitive;
			// we MUST lower-case before matching or Entra silently fails.
			body:         `{"Operations":[{"op":"Add","path":"members","value":[{"value":"u1"}]}]}`,
			wantOps:      []MemberOpJSON{{Op: "add", Members: []string{"u1"}}},
			wantParsedOK: true,
		},
		{
			name: "Entra remove — NON-RFC: member in value, path is bare members",
			// Entra does NOT send a filtered path for remove; the member to drop is
			// in the value array. Parser must accept this.
			body:         `{"Operations":[{"op":"Remove","path":"members","value":[{"value":"u1"}]}]}`,
			wantOps:      []MemberOpJSON{{Op: "remove", Members: []string{"u1"}}},
			wantParsedOK: true,
		},
		{
			name: "Okta/RFC remove — filtered path members[value eq \"x\"]",
			body: `{"Operations":[{"op":"remove","path":"members[value eq \"u1\"]"}]}`,

			wantOps:      []MemberOpJSON{{Op: "remove", Members: []string{"u1"}}},
			wantParsedOK: true,
		},
		{
			name:         "replace whole members set",
			body:         `{"Operations":[{"op":"replace","path":"members","value":[{"value":"u1"},{"value":"u2"}]}]}`,
			wantOps:      []MemberOpJSON{{Op: "replace", Members: []string{"u1", "u2"}}},
			wantParsedOK: true,
		},
		{
			name:         "bare string member values [\"u1\"]",
			body:         `{"Operations":[{"op":"add","path":"members","value":["u1","u2"]}]}`,
			wantOps:      []MemberOpJSON{{Op: "add", Members: []string{"u1", "u2"}}},
			wantParsedOK: true,
		},
		{
			name:         "displayName replace with path",
			body:         `{"Operations":[{"op":"replace","path":"displayName","value":"Engineers"}]}`,
			wantDisplay:  strptr("Engineers"),
			wantParsedOK: true,
		},
		{
			name:         "Entra no-path form — value is attribute map",
			body:         `{"Operations":[{"op":"Replace","value":{"displayName":"Engineers","members":[{"value":"u1"}]}}]}`,
			wantDisplay:  strptr("Engineers"),
			wantOps:      []MemberOpJSON{{Op: "replace", Members: []string{"u1"}}},
			wantParsedOK: true,
		},
		{
			name:         "multiple ops in one PatchOp (add then remove)",
			body:         `{"Operations":[{"op":"add","path":"members","value":[{"value":"u1"}]},{"op":"remove","path":"members[value eq \"u2\"]"}]}`,
			wantOps:      []MemberOpJSON{{Op: "add", Members: []string{"u1"}}, {Op: "remove", Members: []string{"u2"}}},
			wantParsedOK: true,
		},
		{
			name:         "unknown op is ignored, not fatal",
			body:         `{"Operations":[{"op":"noop","path":"members","value":[{"value":"u1"}]}]}`,
			wantOps:      nil,
			wantParsedOK: true,
		},
		{
			// The exact deprovisioning op an IdP sends to empty a group: replace
			// members with an empty array. Must become an explicit clear, not a
			// silent no-op (the HIGH bug).
			name:         "clear members — replace with empty array",
			body:         `{"Operations":[{"op":"replace","path":"members","value":[]}]}`,
			wantOps:      []MemberOpJSON{{Op: "replace", ClearAll: true}},
			wantParsedOK: true,
		},
		{
			// The other RFC-valid full clear: remove members with no filter/value.
			name:         "clear members — remove with no filter",
			body:         `{"Operations":[{"op":"remove","path":"members"}]}`,
			wantOps:      []MemberOpJSON{{Op: "remove", ClearAll: true}},
			wantParsedOK: true,
		},
		{
			name:           "externalId replace with path",
			body:           `{"Operations":[{"op":"replace","path":"externalId","value":"ext-9"}]}`,
			wantExternalID: strptr("ext-9"),
			wantParsedOK:   true,
		},
		{
			name:            "unsupported path -> invalidPath",
			body:            `{"Operations":[{"op":"replace","path":"emails","value":"x@y.com"}]}`,
			wantInvalidPath: true,
			wantParsedOK:    true,
		},
		{
			name:         "malformed JSON fails to parse",
			body:         `{not json`,
			wantParsedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, ok := parseGroupPatch(strings.NewReader(tt.body))
			require.Equal(t, tt.wantParsedOK, ok)
			if !tt.wantParsedOK {
				return
			}
			if tt.wantDisplay == nil {
				assert.Nil(t, patch.DisplayName)
			} else {
				require.NotNil(t, patch.DisplayName)
				assert.Equal(t, *tt.wantDisplay, *patch.DisplayName)
			}
			if tt.wantExternalID == nil {
				assert.Nil(t, patch.ExternalID)
			} else {
				require.NotNil(t, patch.ExternalID)
				assert.Equal(t, *tt.wantExternalID, *patch.ExternalID)
			}
			assert.Equal(t, tt.wantInvalidPath, patch.InvalidPath)
			assert.Equal(t, tt.wantOps, patch.Ops)
		})
	}
}

func TestExtractFilterValue(t *testing.T) {
	assert.Equal(t, "u1", extractFilterValue(`members[value eq "u1"]`))
	assert.Equal(t, "user@x.com", extractFilterValue(`members[value eq "user@x.com"]`))
	assert.Equal(t, "", extractFilterValue(`members`))
	assert.Equal(t, "", extractFilterValue(`members[value eq u1]`))
}
