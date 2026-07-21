package scim

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/refs"
	svcscim "github.com/authorizerdev/authorizer/internal/service/scim"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// scimName is the SCIM complex "name" attribute (subset).
type scimName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

// scimEmail is one entry of the multi-valued "emails" attribute.
type scimEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary,omitempty"`
}

// scimMeta is the resource metadata block (RFC 7643 §3.1). created/lastModified
// are returned:"default"; they are omitempty so User responses (which do not set
// them) are unchanged, while Group responses populate them from stored timestamps.
type scimMeta struct {
	ResourceType string `json:"resourceType"`
	Location     string `json:"location,omitempty"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
}

// scimUserResource is the wire representation of a SCIM User (request + response).
type scimUserResource struct {
	Schemas    []string    `json:"schemas"`
	ID         string      `json:"id,omitempty"`
	ExternalID string      `json:"externalId,omitempty"`
	UserName   string      `json:"userName"`
	Name       scimName    `json:"name"`
	Emails     []scimEmail `json:"emails,omitempty"`
	Active     bool        `json:"active"`
	Meta       scimMeta    `json:"meta"`
}

// scimListResponse is the RFC 7644 §3.4.2 list envelope.
type scimListResponse struct {
	Schemas      []string           `json:"schemas"`
	TotalResults int                `json:"totalResults"`
	StartIndex   int                `json:"startIndex"`
	ItemsPerPage int                `json:"itemsPerPage"`
	Resources    []scimUserResource `json:"Resources"`
}

// toResource maps a stored user to the SCIM wire form. externalId is de-
// namespaced back to the raw IdP value (stored as "<orgID>:<raw>").
func toResource(orgID string, u *schemas.User) scimUserResource {
	email := refs.StringValue(u.Email)
	res := scimUserResource{
		Schemas:  []string{schemaUser},
		ID:       u.ID,
		UserName: email,
		Name: scimName{
			GivenName:  refs.StringValue(u.GivenName),
			FamilyName: refs.StringValue(u.FamilyName),
		},
		Active: u.IsActive,
		Meta:   scimMeta{ResourceType: "User"},
	}
	if email != "" {
		res.Emails = []scimEmail{{Value: email, Primary: true}}
	}
	if u.ExternalID != nil {
		res.ExternalID = strings.TrimPrefix(refs.StringValue(u.ExternalID), orgID+":")
	}
	return res
}

// parseUser decodes a SCIM User request body into the service input. active
// defaults to true when the attribute is absent (SCIM create default).
func parseUser(c *gin.Context) (svcscim.User, bool) {
	body := struct {
		ExternalID string      `json:"externalId"`
		UserName   string      `json:"userName"`
		Name       scimName    `json:"name"`
		Emails     []scimEmail `json:"emails"`
		Active     *bool       `json:"active"`
	}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return svcscim.User{}, false
	}
	userName := strings.TrimSpace(body.UserName)
	if userName == "" {
		// Fall back to the primary/first email as userName (some IdPs omit it).
		for _, e := range body.Emails {
			if e.Value != "" {
				userName = strings.TrimSpace(e.Value)
				break
			}
		}
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	return svcscim.User{
		ExternalID: strings.TrimSpace(body.ExternalID),
		UserName:   userName,
		GivenName:  body.Name.GivenName,
		FamilyName: body.Name.FamilyName,
		Active:     active,
	}, true
}

func (h *Handler) createUser(c *gin.Context) {
	in, ok := parseUser(c)
	if !ok || in.UserName == "" {
		writeError(c, http.StatusBadRequest, "invalidValue", "userName is required")
		return
	}
	user, existed, err := h.Service.CreateUser(c.Request.Context(), h.orgID(c), in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	status := http.StatusCreated
	if existed {
		// Idempotent create (IdP re-POSTed an already-provisioned user): return
		// the existing resource rather than a duplicate.
		status = http.StatusOK
	}
	writeUser(c, status, h.orgID(c), user)
}

func (h *Handler) getUser(c *gin.Context) {
	user, err := h.Service.GetUser(c.Request.Context(), h.orgID(c), c.Param("id"))
	if err != nil {
		mapServiceError(c, err)
		return
	}
	writeUser(c, http.StatusOK, h.orgID(c), user)
}

// listUsers supports only the `userName eq "..."` filter (the IdP dedup probe).
// An unfiltered list returns an empty set — full org enumeration is out of scope.
func (h *Handler) listUsers(c *gin.Context) {
	orgID := h.orgID(c)
	filter := c.Query("filter")
	resp := scimListResponse{
		Schemas:      []string{schemaListResp},
		StartIndex:   1,
		ItemsPerPage: 0,
		Resources:    []scimUserResource{},
	}
	if userName, ok := parseUserNameEq(filter); ok {
		if user, err := h.Service.FindByUserName(c.Request.Context(), orgID, userName); err == nil && user != nil {
			resp.Resources = append(resp.Resources, toResource(orgID, user))
		}
	}
	resp.TotalResults = len(resp.Resources)
	resp.ItemsPerPage = len(resp.Resources)
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) replaceUser(c *gin.Context) {
	in, ok := parseUser(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidValue", "invalid body")
		return
	}
	user, err := h.Service.ReplaceUser(c.Request.Context(), h.orgID(c), c.Param("id"), in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	writeUser(c, http.StatusOK, h.orgID(c), user)
}

// patchUser applies a SCIM PatchOp. ponytail: only the `active` attribute is
// honoured (the deprovision path Okta/Entra drive); other paths are accepted
// and ignored. Full attribute PATCH is a follow-up — use PUT for profile edits.
func (h *Handler) patchUser(c *gin.Context) {
	active, hasActive, ok := parseActivePatch(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidValue", "invalid PatchOp body")
		return
	}
	if !hasActive {
		// Nothing we act on — return the current resource unchanged.
		user, err := h.Service.GetUser(c.Request.Context(), h.orgID(c), c.Param("id"))
		if err != nil {
			mapServiceError(c, err)
			return
		}
		writeUser(c, http.StatusOK, h.orgID(c), user)
		return
	}
	user, err := h.Service.SetActive(c.Request.Context(), h.orgID(c), c.Param("id"), active)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	writeUser(c, http.StatusOK, h.orgID(c), user)
}

// deleteUser maps DELETE to soft-deactivation (active:false) so the user's
// sessions/tokens are revoked but the audit trail and memberships survive.
func (h *Handler) deleteUser(c *gin.Context) {
	_, err := h.Service.SetActive(c.Request.Context(), h.orgID(c), c.Param("id"), false)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeUser(c *gin.Context, status int, orgID string, user *schemas.User) {
	c.Header("Content-Type", contentType)
	c.JSON(status, toResource(orgID, user))
}

// parseUserNameEq extracts X from `userName eq "X"` (case-insensitive operator).
func parseUserNameEq(filter string) (string, bool) {
	f := strings.TrimSpace(filter)
	lower := strings.ToLower(f)
	if !strings.HasPrefix(lower, "username eq ") {
		return "", false
	}
	val := strings.TrimSpace(f[len("username eq "):])
	val = strings.Trim(val, `"`)
	if val == "" {
		return "", false
	}
	return val, true
}

// parseActivePatch reads a SCIM PatchOp and returns the requested active value.
// It tolerates the two shapes IdPs send:
//
//	{"op":"replace","path":"active","value":false}
//	{"op":"replace","value":{"active":false}}
//
// op case is ignored; value may be a bool or the string "true"/"false".
func parseActivePatch(c *gin.Context) (active bool, hasActive bool, ok bool) {
	body := struct {
		Operations []struct {
			Op    string          `json:"op"`
			Path  string          `json:"path"`
			Value json.RawMessage `json:"value"`
		} `json:"Operations"`
	}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return false, false, false
	}
	for _, op := range body.Operations {
		path := strings.ToLower(strings.TrimSpace(op.Path))
		if path == "active" {
			if v, valOK := parseBoolValue(op.Value); valOK {
				active, hasActive = v, true
			}
			continue
		}
		if path == "" {
			// Value is an attribute map, e.g. {"active": false}.
			m := map[string]json.RawMessage{}
			if err := json.Unmarshal(op.Value, &m); err == nil {
				for k, raw := range m {
					if strings.EqualFold(k, "active") {
						if v, valOK := parseBoolValue(raw); valOK {
							active, hasActive = v, true
						}
					}
				}
			}
		}
	}
	return active, hasActive, true
}

// parseBoolValue accepts a JSON bool or a quoted "true"/"false" string.
func parseBoolValue(raw json.RawMessage) (bool, bool) {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}
