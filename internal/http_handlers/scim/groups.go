package scim

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/refs"
	svcscim "github.com/authorizerdev/authorizer/internal/service/scim"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// scimGroupMember is one entry of the multi-valued "members" attribute
// (RFC 7643 §4.2). value is the member's Authorizer user id.
type scimGroupMember struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Display string `json:"display,omitempty"`
}

// scimGroupResource is the wire representation of a SCIM Group (request + response).
type scimGroupResource struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id,omitempty"`
	ExternalID  string            `json:"externalId,omitempty"`
	DisplayName string            `json:"displayName"`
	Members     []scimGroupMember `json:"members"`
	Meta        scimMeta          `json:"meta"`
}

// scimGroupListResponse is the RFC 7644 §3.4.2 list envelope for groups.
type scimGroupListResponse struct {
	Schemas      []string            `json:"schemas"`
	TotalResults int                 `json:"totalResults"`
	StartIndex   int                 `json:"startIndex"`
	ItemsPerPage int                 `json:"itemsPerPage"`
	Resources    []scimGroupResource `json:"Resources"`
}

// toGroupResource maps a stored group + its member ids to the SCIM wire form.
// externalId is de-namespaced back to the raw IdP value (stored "<orgID>:<raw>").
func toGroupResource(orgID string, g *schemas.ScimGroup, memberIDs []string) scimGroupResource {
	res := scimGroupResource{
		Schemas:     []string{schemaGroup},
		ID:          g.ID,
		DisplayName: g.DisplayName,
		Members:     []scimGroupMember{},
		Meta:        scimMeta{ResourceType: "Group"},
	}
	if g.ExternalID != nil {
		res.ExternalID = strings.TrimPrefix(refs.StringValue(g.ExternalID), orgID+":")
	}
	for _, uid := range memberIDs {
		res.Members = append(res.Members, scimGroupMember{Value: uid, Ref: "../Users/" + uid, Type: "User"})
	}
	return res
}

// parseGroup decodes a SCIM Group create/replace body into the service input.
func parseGroup(c *gin.Context) (svcscim.Group, bool) {
	body := struct {
		ExternalID  string          `json:"externalId"`
		DisplayName string          `json:"displayName"`
		Members     json.RawMessage `json:"members"`
	}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return svcscim.Group{}, false
	}
	return svcscim.Group{
		ExternalID:  strings.TrimSpace(body.ExternalID),
		DisplayName: strings.TrimSpace(body.DisplayName),
		Members:     parseMemberValues(body.Members),
	}, true
}

func (h *Handler) createGroup(c *gin.Context) {
	in, ok := parseGroup(c)
	if !ok || in.DisplayName == "" {
		writeError(c, http.StatusBadRequest, "invalidValue", "displayName is required")
		return
	}
	group, existed, err := h.Service.CreateGroup(c.Request.Context(), h.orgID(c), in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	h.writeGroup(c, status, h.orgID(c), group)
}

func (h *Handler) getGroup(c *gin.Context) {
	group, err := h.Service.GetGroup(c.Request.Context(), h.orgID(c), c.Param("id"))
	if err != nil {
		mapServiceError(c, err)
		return
	}
	h.writeGroup(c, http.StatusOK, h.orgID(c), group)
}

// listGroups supports only the `displayName eq "..."` filter (the IdP dedup
// probe), mirroring listUsers. An unfiltered list returns an empty set.
func (h *Handler) listGroups(c *gin.Context) {
	orgID := h.orgID(c)
	resp := scimGroupListResponse{
		Schemas:    []string{schemaListResp},
		StartIndex: 1,
		Resources:  []scimGroupResource{},
	}
	if displayName, ok := parseDisplayNameEq(c.Query("filter")); ok {
		if group, err := h.Service.FindGroupByDisplayName(c.Request.Context(), orgID, displayName); err == nil && group != nil {
			members, _ := h.Service.GroupMembers(c.Request.Context(), orgID, group.ID)
			resp.Resources = append(resp.Resources, toGroupResource(orgID, group, members))
		}
	}
	resp.TotalResults = len(resp.Resources)
	resp.ItemsPerPage = len(resp.Resources)
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) replaceGroup(c *gin.Context) {
	in, ok := parseGroup(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidValue", "invalid body")
		return
	}
	group, err := h.Service.ReplaceGroup(c.Request.Context(), h.orgID(c), c.Param("id"), in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	h.writeGroup(c, http.StatusOK, h.orgID(c), group)
}

func (h *Handler) patchGroup(c *gin.Context) {
	displayName, ops, ok := parseGroupPatch(c.Request.Body)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidValue", "invalid PatchOp body")
		return
	}
	group, err := h.Service.PatchGroup(c.Request.Context(), h.orgID(c), c.Param("id"), displayName, ops)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	h.writeGroup(c, http.StatusOK, h.orgID(c), group)
}

func (h *Handler) deleteGroup(c *gin.Context) {
	if err := h.Service.DeleteGroup(c.Request.Context(), h.orgID(c), c.Param("id")); err != nil {
		mapServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) writeGroup(c *gin.Context, status int, orgID string, group *schemas.ScimGroup) {
	members, _ := h.Service.GroupMembers(c.Request.Context(), orgID, group.ID)
	c.Header("Content-Type", contentType)
	c.JSON(status, toGroupResource(orgID, group, members))
}

// parseDisplayNameEq extracts X from `displayName eq "X"` (case-insensitive op).
func parseDisplayNameEq(filter string) (string, bool) {
	f := strings.TrimSpace(filter)
	lower := strings.ToLower(f)
	const prefix = "displayname eq "
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	val := strings.Trim(strings.TrimSpace(f[len(prefix):]), `"`)
	if val == "" {
		return "", false
	}
	return val, true
}

// parseGroupPatch parses a SCIM Group PatchOp (RFC 7644 §3.5.2) into an optional
// displayName change plus member add/remove/replace ops. It is written for the
// real world, not just the RFC — it accepts every shape Okta and Entra send:
//
//   - op case is normalised (Entra sends "Add"/"Replace"/"Remove").
//   - members add/replace:  {"op":"add","path":"members","value":[{"value":"x"}]}
//   - members remove (Entra, NON-RFC): {"op":"remove","path":"members","value":[{"value":"x"}]}
//     — the member is in `value`, not the path.
//   - members remove (RFC/Okta): {"op":"remove","path":"members[value eq \"x\"]"} —
//     the member is encoded in the filtered path.
//   - no-path form: {"op":"add","value":{"members":[{"value":"x"}]}} / displayName.
//   - member values may be [{"value":"x"}] objects or bare ["x"] strings.
func parseGroupPatch(r io.Reader) (displayName *string, ops []MemberOpJSON, ok bool) {
	body := struct {
		Operations []struct {
			Op    string          `json:"op"`
			Path  string          `json:"path"`
			Value json.RawMessage `json:"value"`
		} `json:"Operations"`
	}{}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return nil, nil, false
	}
	for _, raw := range body.Operations {
		op := strings.ToLower(strings.TrimSpace(raw.Op))
		if op != "add" && op != "remove" && op != "replace" {
			continue
		}
		path := strings.TrimSpace(raw.Path)
		lpath := strings.ToLower(path)
		switch {
		case strings.HasPrefix(lpath, "members["):
			// RFC/Okta filtered path: the member id is inside [value eq "x"].
			if val := extractFilterValue(path); val != "" {
				ops = append(ops, MemberOpJSON{Op: op, Members: []string{val}})
			}
		case lpath == "members":
			// Entra remove (member in value) and add/replace all land here.
			if members := parseMemberValues(raw.Value); len(members) > 0 {
				ops = append(ops, MemberOpJSON{Op: op, Members: members})
			}
		case lpath == "displayname":
			if dn, dok := parseStringValue(raw.Value); dok {
				displayName = &dn
			}
		case lpath == "":
			// No path: value is an attribute map, e.g.
			// {"members":[...], "displayName":"..."}.
			m := map[string]json.RawMessage{}
			if err := json.Unmarshal(raw.Value, &m); err != nil {
				continue
			}
			for k, v := range m {
				switch strings.ToLower(strings.TrimSpace(k)) {
				case "members":
					if members := parseMemberValues(v); len(members) > 0 {
						ops = append(ops, MemberOpJSON{Op: op, Members: members})
					}
				case "displayname":
					if dn, dok := parseStringValue(v); dok {
						displayName = &dn
					}
				}
			}
		}
	}
	return displayName, ops, true
}

// MemberOpJSON is the transport twin of svcscim.MemberOp. Kept in the handler so
// the parser stays transport-side; converted to the service type by the handler.
type MemberOpJSON = svcscim.MemberOp

// parseMemberValues extracts member ids from a SCIM `members` value, tolerating
// both [{"value":"x"}] complex entries and bare ["x"] strings, plus a single
// {"value":"x"} object.
func parseMemberValues(raw json.RawMessage) []string {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var objs []struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &objs); err == nil {
		out := make([]string, 0, len(objs))
		for _, o := range objs {
			if v := strings.TrimSpace(o.Value); v != "" {
				out = append(out, v)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		out := make([]string, 0, len(strs))
		for _, s := range strs {
			if v := strings.TrimSpace(s); v != "" {
				out = append(out, v)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var one struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &one); err == nil {
		if v := strings.TrimSpace(one.Value); v != "" {
			return []string{v}
		}
	}
	return nil
}

// parseStringValue reads a SCIM value that should be a plain string (displayName).
func parseStringValue(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s != "" {
			return s, true
		}
	}
	return "", false
}

// extractFilterValue pulls X out of a `members[value eq "X"]` filtered path.
func extractFilterValue(path string) string {
	open := strings.Index(path, "[")
	closeIdx := strings.LastIndex(path, "]")
	if open < 0 || closeIdx < 0 || closeIdx <= open {
		return ""
	}
	inner := path[open+1 : closeIdx]
	q1 := strings.Index(inner, `"`)
	q2 := strings.LastIndex(inner, `"`)
	if q1 < 0 || q2 <= q1 {
		return ""
	}
	return inner[q1+1 : q2]
}
