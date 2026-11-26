package service

import "context"

// Conflict probes, used after a failed create.
//
// Services pre-check uniqueness before writing, but the pre-check and the write
// are not atomic, and some writes carry identifiers the pre-check does not
// cover (e.g. the phone number on an email signup). When the write then fails,
// each backend reports the conflict in its own words — a SQL unique-index
// violation, a NoSQL "already exists" error — and all of them arrive as an
// unclassified storage error, which every transport masks as a 500.
//
// Rather than parse thirteen backends' error text, the service asks storage
// again: if the identifier is now taken, the failure was a conflict and is
// reported as a typed error (409 / the caller's documented wording); otherwise
// the original error is returned and stays masked as internal. A lookup error
// during the probe counts as "not taken", so an outage is never misreported as
// a conflict.
//
// The pattern is applied at each failed create: SignUp and InviteMembers use
// userIdentifierTaken below; CreateOrganization and AddOrgMember probe inline
// with GetOrganizationByName / GetOrgMembership (admin_organizations.go), the
// same getters their pre-checks use.
//
// Known edge case, accepted: a write that was applied but still reported an
// error (e.g. a Cassandra/DynamoDB timeout after the write landed) is probed
// as "taken" and reported as a conflict, although the row is this request's
// own. The caller sees a 409/"already exists" for a resource that does now
// exist, which is recoverable; the alternative — trusting an ambiguous write
// error — would report a 500 for the same state.

// userIdentifierTaken reports whether a user already holds email or phone.
func (p *provider) userIdentifierTaken(ctx context.Context, email, phone string) bool {
	if email != "" {
		if u, err := p.StorageProvider.GetUserByEmail(ctx, email); err == nil && u != nil {
			return true
		}
	}
	if phone != "" {
		if u, err := p.StorageProvider.GetUserByPhoneNumber(ctx, phone); err == nil && u != nil {
			return true
		}
	}
	return false
}
