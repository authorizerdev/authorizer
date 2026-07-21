package scim

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Group is the transport-neutral SCIM group projection the handler maps to/from
// SCIM JSON. Members are Authorizer user ids (the SCIM member "value").
type Group struct {
	ExternalID  string
	DisplayName string
	Members     []string
}

// MemberOp is a single parsed SCIM PatchOp entry over the `members` attribute.
// Op is lower-cased (RFC 7644 §3.5.2 says op is case-insensitive; real IdPs send
// mixed case). Members are Authorizer user ids extracted from either the RFC/Okta
// filtered-path shape or the Entra value-array shape (see the handler's parser).
type MemberOp struct {
	Op      string // "add" | "remove" | "replace"
	Members []string
}

// groupMemberRelation is the FGA relation binding a user (or nested group) to a
// group. Model: `type group { define member: [user, group#member] }`.
const groupMemberRelation = "member"

// groupObject builds the org-namespaced FGA object id for a group. The org
// prefix is the tenant-isolation boundary: a group object always carries the
// org it belongs to, so a cross-tenant read can never match this org's prefix.
func groupObject(orgID, groupID string) string {
	return "group:" + orgID + "/" + groupID
}

func userSubject(userID string) string {
	return "user:" + userID
}

// requireGroup is the H6 isolation gate for groups: a group is visible/mutable
// through a SCIM connection only if it belongs to that connection's org. A
// cross-org id therefore returns ErrNotFound, never another org's group.
func (p *provider) requireGroup(ctx context.Context, orgID, groupID string) (*schemas.ScimGroup, error) {
	group, err := p.StorageProvider.GetScimGroupByID(ctx, groupID)
	if err != nil || group == nil || group.OrgID != orgID {
		return nil, ErrNotFound
	}
	return group, nil
}

// CreateGroup provisions a group into the org (idempotent by displayName/externalId).
func (p *provider) CreateGroup(ctx context.Context, orgID string, in Group) (*schemas.ScimGroup, bool, error) {
	log := p.Log.With().Str("func", "scim.CreateGroup").Str("org_id", orgID).Logger()
	if p.AuthzEngine == nil {
		return nil, false, ErrGroupsUnavailable
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return nil, false, ErrInvalid
	}

	// Dedup: same displayName already provisioned into this org → idempotent.
	if existing, err := p.StorageProvider.GetScimGroupByOrgAndDisplayName(ctx, orgID, displayName); err == nil && existing != nil {
		log.Debug().Msg("dedup by displayName within org")
		if err := p.syncMembers(ctx, orgID, existing.ID, in.Members, nil); err != nil {
			return nil, false, err
		}
		return existing, true, nil
	}

	now := time.Now().Unix()
	newGroup := &schemas.ScimGroup{
		OrgID:       orgID,
		DisplayName: displayName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if in.ExternalID != "" {
		nsExt := namespacedExternalID(orgID, in.ExternalID)
		newGroup.ExternalID = &nsExt
	}
	created, err := p.StorageProvider.AddScimGroup(ctx, newGroup)
	if err != nil {
		log.Debug().Err(err).Msg("failed to add scim group")
		return nil, false, err
	}
	if err := p.syncMembers(ctx, orgID, created.ID, in.Members, nil); err != nil {
		return nil, false, err
	}
	return created, false, nil
}

func (p *provider) GetGroup(ctx context.Context, orgID, groupID string) (*schemas.ScimGroup, error) {
	return p.requireGroup(ctx, orgID, groupID)
}

func (p *provider) FindGroupByDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, nil
	}
	group, err := p.StorageProvider.GetScimGroupByOrgAndDisplayName(ctx, orgID, displayName)
	if err != nil || group == nil {
		return nil, nil
	}
	return group, nil
}

func (p *provider) ReplaceGroup(ctx context.Context, orgID, groupID string, in Group) (*schemas.ScimGroup, error) {
	if p.AuthzEngine == nil {
		return nil, ErrGroupsUnavailable
	}
	group, err := p.requireGroup(ctx, orgID, groupID)
	if err != nil {
		return nil, err
	}
	if dn := strings.TrimSpace(in.DisplayName); dn != "" && dn != group.DisplayName {
		group.DisplayName = dn
		group.UpdatedAt = time.Now().Unix()
		if group, err = p.StorageProvider.UpdateScimGroup(ctx, group); err != nil {
			return nil, err
		}
	}
	// PUT replaces the whole membership set: desired = in.Members exactly.
	if err := p.replaceMembers(ctx, orgID, groupID, in.Members); err != nil {
		return nil, err
	}
	return group, nil
}

func (p *provider) PatchGroup(ctx context.Context, orgID, groupID string, displayName *string, ops []MemberOp) (*schemas.ScimGroup, error) {
	if p.AuthzEngine == nil {
		return nil, ErrGroupsUnavailable
	}
	group, err := p.requireGroup(ctx, orgID, groupID)
	if err != nil {
		return nil, err
	}
	if displayName != nil {
		if dn := strings.TrimSpace(*displayName); dn != "" && dn != group.DisplayName {
			group.DisplayName = dn
			group.UpdatedAt = time.Now().Unix()
			if group, err = p.StorageProvider.UpdateScimGroup(ctx, group); err != nil {
				return nil, err
			}
		}
	}
	for _, op := range ops {
		switch op.Op {
		case "add":
			if err := p.syncMembers(ctx, orgID, groupID, op.Members, nil); err != nil {
				return nil, err
			}
		case "remove":
			if err := p.syncMembers(ctx, orgID, groupID, nil, op.Members); err != nil {
				return nil, err
			}
		case "replace":
			// replace on `members` sets membership to exactly this list.
			if err := p.replaceMembers(ctx, orgID, groupID, op.Members); err != nil {
				return nil, err
			}
		}
	}
	return group, nil
}

func (p *provider) DeleteGroup(ctx context.Context, orgID, groupID string) error {
	group, err := p.requireGroup(ctx, orgID, groupID)
	if err != nil {
		return err
	}
	if p.AuthzEngine != nil {
		// Drop every membership tuple (subjects → group) so no member resolves
		// through the deleted group.
		//
		// ponytail: we do NOT hunt down role→group binding tuples
		// (role:org/r#assignee@group:org/g#member) here — OpenFGA Read needs an
		// object filter, so a by-subject sweep isn't a clean call. It is safe to
		// leave them: group ids are UUIDs and never reused, so once the member
		// tuples above are gone the dangling binding resolves to nobody. An admin
		// removes the binding via _fga_delete_tuples if desired.
		if members, err := p.readMemberSubjects(ctx, orgID, groupID); err == nil && len(members) > 0 {
			tuples := make([]engine.TupleKey, 0, len(members))
			for _, subj := range members {
				tuples = append(tuples, engine.TupleKey{User: subj, Relation: groupMemberRelation, Object: groupObject(orgID, groupID)})
			}
			_ = p.AuthzEngine.DeleteTuples(ctx, tuples)
		}
	}
	return p.StorageProvider.DeleteScimGroup(ctx, group)
}

// GroupMembers returns the direct member user ids of an org's group.
func (p *provider) GroupMembers(ctx context.Context, orgID, groupID string) ([]string, error) {
	if _, err := p.requireGroup(ctx, orgID, groupID); err != nil {
		return nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil
	}
	subjects, err := p.readMemberSubjects(ctx, orgID, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(subjects))
	for _, s := range subjects {
		// Only surface direct user members (skip nested group usersets) for the
		// SCIM `members` response.
		if uid, ok := strings.CutPrefix(s, "user:"); ok {
			out = append(out, uid)
		}
	}
	return out, nil
}

// syncMembers adds `add` and removes `remove` member user ids on a group,
// idempotently. Every added user MUST be a member of the org (H6 cross-tenant
// gate): a user id belonging to another org is silently skipped, never written.
func (p *provider) syncMembers(ctx context.Context, orgID, groupID string, add, remove []string) error {
	log := p.Log.With().Str("func", "scim.syncMembers").Str("org_id", orgID).Logger()
	existing, err := p.readMemberSubjects(ctx, orgID, groupID)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(existing))
	for _, s := range existing {
		present[s] = true
	}
	obj := groupObject(orgID, groupID)

	var writes, deletes []engine.TupleKey
	for _, uid := range dedupTrim(add) {
		subj := userSubject(uid)
		if present[subj] {
			continue
		}
		// Cross-tenant gate: only org members may be added to this org's group.
		if _, err := p.StorageProvider.GetOrgMembership(ctx, orgID, uid); err != nil {
			log.Debug().Str("user_id", uid).Msg("skipping member add: not a member of this org")
			continue
		}
		writes = append(writes, engine.TupleKey{User: subj, Relation: groupMemberRelation, Object: obj})
		present[subj] = true
	}
	for _, uid := range dedupTrim(remove) {
		subj := userSubject(uid)
		if !present[subj] {
			continue
		}
		deletes = append(deletes, engine.TupleKey{User: subj, Relation: groupMemberRelation, Object: obj})
		present[subj] = false
	}
	if len(writes) > 0 {
		if err := p.AuthzEngine.WriteTuples(ctx, writes); err != nil {
			return err
		}
	}
	if len(deletes) > 0 {
		if err := p.AuthzEngine.DeleteTuples(ctx, deletes); err != nil {
			return err
		}
	}
	return nil
}

// replaceMembers sets a group's membership to exactly `desired` (the PUT / SCIM
// replace semantics): writes the missing, deletes the surplus.
func (p *provider) replaceMembers(ctx context.Context, orgID, groupID string, desired []string) error {
	existing, err := p.readMemberSubjects(ctx, orgID, groupID)
	if err != nil {
		return err
	}
	want := make(map[string]bool)
	for _, uid := range dedupTrim(desired) {
		want[userSubject(uid)] = true
	}
	var remove []string
	for _, s := range existing {
		if uid, ok := strings.CutPrefix(s, "user:"); ok && !want[s] {
			remove = append(remove, uid)
		}
	}
	return p.syncMembers(ctx, orgID, groupID, desired, remove)
}

// readMemberSubjects reads the direct member tuple subjects of a group
// (user:<id> and any group:<...>#member usersets), paginating through the store.
func (p *provider) readMemberSubjects(ctx context.Context, orgID, groupID string) ([]string, error) {
	obj := groupObject(orgID, groupID)
	var subjects []string
	token := ""
	for {
		page, err := p.AuthzEngine.ReadTuples(ctx, engine.ReadTuplesFilter{
			Relation:          groupMemberRelation,
			Object:            obj,
			PageSize:          100,
			ContinuationToken: token,
		})
		if err != nil {
			return nil, err
		}
		for _, t := range page.Tuples {
			subjects = append(subjects, t.User)
		}
		if page.ContinuationToken == "" {
			break
		}
		token = page.ContinuationToken
	}
	return subjects, nil
}

// dedupTrim trims, drops empties, and de-duplicates a slice of ids.
func dedupTrim(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
