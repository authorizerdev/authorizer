package scim

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// listUsers evaluates a single-term SCIM filter (RFC 7644 §3.4.2.2) — operators
// eq/ne/co/sw/pr over userName, emails.value, name.givenName, name.familyName,
// active, externalId — against the org's members, honouring startIndex/count
// pagination. An unfiltered list returns an empty set (full org enumeration is
// out of scope); a filter the parser does not recognize is a 400 invalidFilter
// (never an empty 200, which a connector reads as "no such user" and duplicates).
func (h *Handler) listUsers(c *gin.Context) {
	orgID := h.orgID(c)
	resp := scimListResponse{
		Schemas:    []string{schemaListResp},
		StartIndex: 1,
		Resources:  []scimUserResource{},
	}
	filter := strings.TrimSpace(c.Query("filter"))
	if filter == "" {
		resp.ItemsPerPage = 0
		c.Header("Content-Type", contentType)
		c.JSON(http.StatusOK, resp)
		return
	}
	f, ok := parseUserFilter(filter)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidFilter", "unsupported filter expression")
		return
	}
	users, err := h.Service.ListUsers(c.Request.Context(), orgID, f)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	all := make([]scimUserResource, 0, len(users))
	for _, u := range users {
		all = append(all, toResource(orgID, u))
	}
	// RFC 7644 §3.4.2.4 pagination: startIndex is 1-based, count caps the page.
	startIndex := queryInt(c, "startIndex", 1)
	if startIndex < 1 {
		startIndex = 1
	}
	count := queryInt(c, "count", len(all))
	if count < 0 {
		count = 0
	}
	from := startIndex - 1
	if from > len(all) {
		from = len(all)
	}
	to := from + count
	if to > len(all) {
		to = len(all)
	}
	resp.TotalResults = len(all)
	resp.StartIndex = startIndex
	resp.Resources = all[from:to]
	resp.ItemsPerPage = len(resp.Resources)
	c.Header("Content-Type", contentType)
	c.JSON(http.StatusOK, resp)
}

// queryInt reads a non-negative integer query param, returning def when absent
// or unparseable.
func queryInt(c *gin.Context, key string, def int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
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

// patchUser applies a SCIM PatchOp (RFC 7644 §3.5.2). It honours replace/add on
// active, name.givenName, name.familyName, userName/emails (→ email),
// phoneNumbers (→ phone), and externalId, in both the path-qualified and the
// no-path attribute-map shapes. Paths the server does not model (e.g. enterprise
// extension attributes) are ignored rather than 400'd, so a connector that also
// syncs manager/department does not fail the whole request. An empty patch
// returns the current resource unchanged (no event).
func (h *Handler) patchUser(c *gin.Context) {
	patch, ok := parseUserPatch(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalidSyntax", "invalid PatchOp body")
		return
	}
	user, err := h.Service.PatchUser(c.Request.Context(), h.orgID(c), c.Param("id"), patch)
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

// userFilterAttrs maps a lower-cased SCIM attribute path to its canonical name
// (the form svcscim.UserFilter / the service expect). Only the attributes SCIM
// directory sync actually filters on are supported.
var userFilterAttrs = map[string]string{
	"username":        "userName",
	"emails.value":    "emails.value",
	"name.givenname":  "name.givenName",
	"name.familyname": "name.familyName",
	"active":          "active",
	"externalid":      "externalId",
}

// parseUserFilter parses a single-term SCIM filter `attribute op [value]` (RFC
// 7644 §3.4.2.2) into a svcscim.UserFilter. Supported operators: eq, ne, co, sw,
// pr. Compound filters (and/or/not), value-path filters (emails[type eq ...]),
// and ordering operators (gt/lt/…) are deliberately unsupported — real user
// directory-sync filters are single-term — and return ok=false so the handler
// answers 400 invalidFilter rather than silently matching nothing.
func parseUserFilter(filter string) (svcscim.UserFilter, bool) {
	f := strings.TrimSpace(filter)
	// Reject compound expressions outright.
	if lf := strings.ToLower(f); strings.Contains(lf, " and ") || strings.Contains(lf, " or ") || strings.HasPrefix(lf, "not ") || strings.ContainsAny(f, "()[]") {
		return svcscim.UserFilter{}, false
	}
	attrTok, rest, ok := strings.Cut(f, " ")
	if !ok {
		return svcscim.UserFilter{}, false
	}
	attr, ok := userFilterAttrs[strings.ToLower(strings.TrimSpace(attrTok))]
	if !ok {
		return svcscim.UserFilter{}, false
	}
	rest = strings.TrimSpace(rest)
	opTok, valTok, hasVal := strings.Cut(rest, " ")
	op := strings.ToLower(strings.TrimSpace(opTok))

	switch op {
	case "pr":
		// present takes no value.
		if strings.TrimSpace(valTok) != "" {
			return svcscim.UserFilter{}, false
		}
		return svcscim.UserFilter{Attribute: attr, Operator: "pr"}, true
	case "eq", "ne", "co", "sw":
		if !hasVal {
			return svcscim.UserFilter{}, false
		}
		// co/sw are meaningless on the boolean `active` attribute.
		if attr == "active" && (op == "co" || op == "sw") {
			return svcscim.UserFilter{}, false
		}
		val := unquoteFilterValue(valTok)
		if val == "" {
			return svcscim.UserFilter{}, false
		}
		return svcscim.UserFilter{Attribute: attr, Operator: op, Value: val}, true
	default:
		return svcscim.UserFilter{}, false
	}
}

// parseUserPatch reads a SCIM User PatchOp into a svcscim.UserPatch. It handles
// replace/add ops (single-valued attributes treat add ≡ replace) in both the
// path-qualified shape:
//
//	{"op":"replace","path":"name.givenName","value":"Jonathan"}
//	{"op":"replace","path":"active","value":false}
//	{"op":"replace","path":"emails[type eq \"work\"].value","value":"a@b.com"}
//
// and the no-path attribute-map shape:
//
//	{"op":"replace","value":{"name":{"givenName":"Jonathan"},"active":false}}
//
// op case is ignored. remove ops and unmodelled paths are silently skipped.
// ok=false only for a malformed body.
func parseUserPatch(c *gin.Context) (svcscim.UserPatch, bool) {
	body := struct {
		Operations []struct {
			Op    string          `json:"op"`
			Path  string          `json:"path"`
			Value json.RawMessage `json:"value"`
		} `json:"Operations"`
	}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return svcscim.UserPatch{}, false
	}
	var patch svcscim.UserPatch
	for _, op := range body.Operations {
		verb := strings.ToLower(strings.TrimSpace(op.Op))
		if verb != "replace" && verb != "add" {
			continue
		}
		lpath := strings.ToLower(strings.TrimSpace(op.Path))
		switch {
		case lpath == "active":
			if v, ok := parseBoolValue(op.Value); ok {
				patch.Active = &v
			}
		case lpath == "name.givenname":
			setStrPatch(&patch.GivenName, op.Value)
		case lpath == "name.familyname":
			setStrPatch(&patch.FamilyName, op.Value)
		case lpath == "username":
			setStrPatch(&patch.Email, op.Value)
		case lpath == "externalid":
			setStrPatch(&patch.ExternalID, op.Value)
		case lpath == "emails", strings.HasPrefix(lpath, "emails["):
			if email, ok := emailFromValue(op.Value); ok {
				patch.Email = &email
			}
		case lpath == "phonenumber", lpath == "phonenumbers", strings.HasPrefix(lpath, "phonenumbers["):
			if phone, ok := phoneFromValue(op.Value); ok {
				patch.PhoneNumber = &phone
			}
		case lpath == "":
			applyNoPathUserPatch(&patch, op.Value)
		default:
			// Unmodelled path (e.g. enterprise extension) — ignore, don't 400.
		}
	}
	return patch, true
}

// applyNoPathUserPatch reads the no-path attribute-map value into the patch.
func applyNoPathUserPatch(patch *svcscim.UserPatch, raw json.RawMessage) {
	body := struct {
		UserName   *string         `json:"userName"`
		ExternalID *string         `json:"externalId"`
		Active     json.RawMessage `json:"active"`
		Name       *struct {
			GivenName  *string `json:"givenName"`
			FamilyName *string `json:"familyName"`
		} `json:"name"`
		Emails       json.RawMessage `json:"emails"`
		PhoneNumbers json.RawMessage `json:"phoneNumbers"`
	}{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return
	}
	if body.Name != nil {
		if body.Name.GivenName != nil {
			gn := strings.TrimSpace(*body.Name.GivenName)
			patch.GivenName = &gn
		}
		if body.Name.FamilyName != nil {
			fn := strings.TrimSpace(*body.Name.FamilyName)
			patch.FamilyName = &fn
		}
	}
	if body.UserName != nil {
		un := strings.TrimSpace(*body.UserName)
		patch.Email = &un
	}
	if len(body.Emails) > 0 {
		if email, ok := emailFromValue(body.Emails); ok {
			patch.Email = &email
		}
	}
	if len(body.PhoneNumbers) > 0 {
		if phone, ok := phoneFromValue(body.PhoneNumbers); ok {
			patch.PhoneNumber = &phone
		}
	}
	if body.ExternalID != nil {
		ext := strings.TrimSpace(*body.ExternalID)
		patch.ExternalID = &ext
	}
	if len(body.Active) > 0 {
		if v, ok := parseBoolValue(body.Active); ok {
			patch.Active = &v
		}
	}
}

// setStrPatch decodes a JSON string value and, if non-empty after trim, points
// dst at it.
func setStrPatch(dst **string, raw json.RawMessage) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return
	}
	s = strings.TrimSpace(s)
	*dst = &s
}

// emailFromValue extracts the primary email from a SCIM `emails` PATCH value,
// tolerating a plain string, an array of {value,type,primary}, or a single such
// object. Prefers primary:true, then type=="work", then the first non-empty.
func emailFromValue(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s = strings.TrimSpace(s); s != "" {
			return s, true
		}
		return "", false
	}
	pick := func(emails []scimEmail) (string, bool) {
		first := ""
		for _, e := range emails {
			v := strings.TrimSpace(e.Value)
			if v == "" {
				continue
			}
			if e.Primary {
				return v, true
			}
			if first == "" {
				first = v
			}
		}
		if first != "" {
			return first, true
		}
		return "", false
	}
	var arr []scimEmail
	if err := json.Unmarshal(raw, &arr); err == nil {
		return pick(arr)
	}
	var one scimEmail
	if err := json.Unmarshal(raw, &one); err == nil {
		if v := strings.TrimSpace(one.Value); v != "" {
			return v, true
		}
	}
	return "", false
}

// phoneFromValue extracts a phone number from a SCIM `phoneNumbers` PATCH value,
// tolerating a plain string, an array of {value}, or a single {value} object.
func phoneFromValue(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s = strings.TrimSpace(s); s != "" {
			return s, true
		}
		return "", false
	}
	type phone struct {
		Value string `json:"value"`
	}
	var arr []phone
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, e := range arr {
			if v := strings.TrimSpace(e.Value); v != "" {
				return v, true
			}
		}
	}
	var one phone
	if err := json.Unmarshal(raw, &one); err == nil {
		if v := strings.TrimSpace(one.Value); v != "" {
			return v, true
		}
	}
	return "", false
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
