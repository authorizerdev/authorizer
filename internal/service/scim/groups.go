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
	// ClearAll marks an unfiltered full-membership clear: a `remove` op whose
	// `members` key is present with an empty/absent value (RFC 7644 §3.5.2 — the
	// deprovisioning shape an IdP sends to empty a group). Without this flag such
	// an op carries no member ids and would be a silent no-op. (A `replace` with
	// an empty set already clears via replaceMembers, so ClearAll is only read on
	// `remove`.)
	ClearAll bool
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

// ensureDisplayNameFree returns ErrGroupConflict when another group in the org
// (id != exceptID) already uses displayName. This is the create/rename
// uniqueness gate — displayName uniqueness within an org is service-enforced,
// not a DB constraint, so it MUST be checked on every path that sets a name
// (create AND rename), or two rows could share a name and the LIMIT 1 dedup
// lookup would then resolve arbitrarily. Pass exceptID="" on create.
func (p *provider) ensureDisplayNameFree(ctx context.Context, orgID, displayName, exceptID string) error {
	if existing, err := p.StorageProvider.GetScimGroupByOrgAndDisplayName(ctx, orgID, displayName); err == nil && existing != nil && existing.ID != exceptID {
		return ErrGroupConflict
	}
	return nil
}

// CreateGroup provisions a group into the org. externalId is the preferred
// correlation key: a repeat carrying an externalId that already identifies a
// group is idempotent (adopts a rename, syncs members, existed=true). A create
// that instead clashes on displayName with no matching externalId is a
// uniqueness conflict (ErrGroupConflict → 409), never a silent 200.
func (p *provider) CreateGroup(ctx context.Context, orgID string, in Group) (*schemas.ScimGroup, bool, error) {
	log := p.Log.With().Str("func", "scim.CreateGroup").Str("org_id", orgID).Logger()
	if p.AuthzEngine == nil {
		return nil, false, ErrGroupsUnavailable
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return nil, false, ErrInvalid
	}

	// Dedup #1 (correlation key): the same externalId already identifies a group
	// in this org → the same logical group. Idempotent: adopt a rename from the
	// IdP and sync members, rather than creating a duplicate row.
	if in.ExternalID != "" {
		if existing, err := p.StorageProvider.GetScimGroupByOrgAndExternalID(ctx, orgID, in.ExternalID); err == nil && existing != nil {
			log.Debug().Msg("dedup by external_id within org")
			if displayName != existing.DisplayName {
				if err := p.ensureDisplayNameFree(ctx, orgID, displayName, existing.ID); err != nil {
					return nil, false, err
				}
				existing.DisplayName = displayName
				existing.UpdatedAt = time.Now().Unix()
				updated, uErr := p.StorageProvider.UpdateScimGroup(ctx, existing)
				if uErr != nil {
					return nil, false, uErr
				}
				existing = updated
			}
			if err := p.syncMembers(ctx, orgID, existing.ID, in.Members, nil); err != nil {
				return nil, false, err
			}
			return existing, true, nil
		}
	}

	// Dedup #2 (uniqueness): a group with this displayName already exists in the
	// org and no externalId matched it → RFC 7644 §3.3 uniqueness conflict (409),
	// not a silent idempotent 200.
	if existing, err := p.StorageProvider.GetScimGroupByOrgAndDisplayName(ctx, orgID, displayName); err == nil && existing != nil {
		log.Debug().Msg("displayName already exists in org")
		return nil, false, ErrGroupConflict
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
	changed := false
	if dn := strings.TrimSpace(in.DisplayName); dn != "" && dn != group.DisplayName {
		if err := p.ensureDisplayNameFree(ctx, orgID, dn, group.ID); err != nil {
			return nil, err
		}
		group.DisplayName = dn
		changed = true
	}
	// externalId is an updatable correlation key. Only set it when the payload
	// carries one — an absent externalId on PUT does not clear a stored value
	// (many connectors omit it on updates).
	if ext := strings.TrimSpace(in.ExternalID); ext != "" {
		nsExt := namespacedExternalID(orgID, ext)
		if group.ExternalID == nil || *group.ExternalID != nsExt {
			group.ExternalID = &nsExt
			changed = true
		}
	}
	if changed {
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

func (p *provider) PatchGroup(ctx context.Context, orgID, groupID string, displayName, externalID *string, ops []MemberOp) (*schemas.ScimGroup, error) {
	if p.AuthzEngine == nil {
		return nil, ErrGroupsUnavailable
	}
	group, err := p.requireGroup(ctx, orgID, groupID)
	if err != nil {
		return nil, err
	}
	changed := false
	if displayName != nil {
		if dn := strings.TrimSpace(*displayName); dn != "" && dn != group.DisplayName {
			if err := p.ensureDisplayNameFree(ctx, orgID, dn, group.ID); err != nil {
				return nil, err
			}
			group.DisplayName = dn
			changed = true
		}
	}
	if externalID != nil {
		if ext := strings.TrimSpace(*externalID); ext != "" {
			nsExt := namespacedExternalID(orgID, ext)
			if group.ExternalID == nil || *group.ExternalID != nsExt {
				group.ExternalID = &nsExt
				changed = true
			}
		}
	}
	if changed {
		group.UpdatedAt = time.Now().Unix()
		if group, err = p.StorageProvider.UpdateScimGroup(ctx, group); err != nil {
			return nil, err
		}
	}
	for _, op := range ops {
		switch op.Op {
		case "add":
			if err := p.syncMembers(ctx, orgID, groupID, op.Members, nil); err != nil {
				return nil, err
			}
		case "remove":
			if op.ClearAll {
				// Unfiltered "remove members" empties the whole group — the exact
				// deprovisioning op an IdP sends. Desired set is empty → remove all.
				if err := p.replaceMembers(ctx, orgID, groupID, nil); err != nil {
					return nil, err
				}
			} else if err := p.syncMembers(ctx, orgID, groupID, nil, op.Members); err != nil {
				return nil, err
			}
		case "replace":
			// replace on `members` sets membership to exactly this list (an empty
			// list clears every member — a legitimate full-clear, not a no-op).
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
