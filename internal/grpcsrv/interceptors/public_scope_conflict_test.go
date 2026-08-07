package interceptors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/authorizerdev/authorizer/internal/delegatedscope"
)

// TestNoPublicMethodCarriesADelegatedScope closes a latent footgun rather than a
// live bug.
//
// A method marked `(authorizer.v1.public) = true` returns from the auth
// interceptor BEFORE enforceDelegatedScope runs, so its delegated-scope
// requirement — if it has one — is never checked on gRPC/REST/MCP. GraphQL
// gates every field, so the same operation would be enforced there and silently
// unenforced here.
//
// Today exactly one method (Meta) is both reachable publicly and present in the
// scope table, and it is read-only, so nothing is exposed. The hazard is the
// next write method that gets a public fast-path added for a good local reason,
// with no signal that it just dropped its scope check on three transports.
//
// Asserted in CI rather than at boot deliberately: this is a property of the
// proto definitions and the scope table, both fixed at compile time, so the
// failure belongs before the binary ships — not in a startup path that an
// operator discovers at 3am.
func TestNoPublicMethodCarriesADelegatedScope(t *testing.T) {
	t.Parallel()

	type conflict struct {
		method string
		scope  string
	}
	var conflicts []conflict

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				if !isPublicMethod(m) {
					continue
				}
				scope, ok := delegatedscope.RequiredForGRPC(string(m.Name()))
				if !ok {
					continue
				}
				// Meta is the known, reviewed exception: read-only, and its
				// scope entry exists for the GraphQL surface.
				if string(m.Name()) == "Meta" {
					continue
				}
				conflicts = append(conflicts, conflict{
					method: string(svc.FullName()) + "/" + string(m.Name()),
					scope:  scope,
				})
			}
		}
		return true
	})

	assert.Empty(t, conflicts,
		"these methods are marked public AND require a delegated scope, so the scope is enforced on "+
			"GraphQL but skipped on gRPC/REST/MCP. Either drop the `public` option or remove the "+
			"delegated-scope entry — do not leave them disagreeing: %+v", conflicts)
}
