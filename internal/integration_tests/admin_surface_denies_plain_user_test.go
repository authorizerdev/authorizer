package integration_tests

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestAdminSurfaceDeniesPlainUser sweeps EVERY AuthorizerAdminService RPC over
// both wire surfaces with an ordinary authenticated user and asserts each one
// is denied.
//
// This guards the exact regression the interceptor change makes possible. The
// gRPC auth interceptor used to reject every non-super-admin before any admin
// handler ran, so a handler that forgot its own authorization check was still
// unreachable. It no longer does — an org-admin has to reach the service layer
// for org-scoped operations to work at all (see TestSurfaceConformanceOrgAdmin).
// Each admin method is therefore responsible for its own gate now, and this
// test checks that responsibility is actually met, method by method, on the
// real transports.
//
// The caller is a freshly signed-up user: authenticated, member of no
// organization, holder of no admin credential. Nothing on the admin surface may
// serve them — neither the platform-wide operations nor the org-scoped ones.
//
// The probe payload matters. Requests are filled and then checked against the
// SAME protovalidate rules the server's validate interceptor enforces, because
// that interceptor sits between auth and the handler: a request that fails
// validation is rejected with InvalidArgument having never reached the
// authorization gate, so asserting on it would prove nothing. Any method whose
// payload cannot be made valid generically is reported and skipped rather than
// asserted on — see the coverage floor and the t.Log at the end.
//
// TestAdminMethodsAreGated (internal/service) proves the same property by
// parsing the source. Both are kept deliberately: the AST test names the
// offending method the moment it is written, and this one proves the gate is
// actually reached at runtime through the interceptor, handler, and gateway.
func TestAdminSurfaceDeniesPlainUser(t *testing.T) {
	s := newSurfaces(t)

	// A plain authenticated user: no admin secret, no org membership.
	email := "plain_" + uuid.NewString() + "@authorizer.dev"
	const pw = "Password@123"
	signup, err := s.ts.GraphQLProvider.SignUp(s.gqlCtx, &model.SignUpRequest{
		Email: &email, Password: pw, ConfirmPassword: pw,
	})
	require.NoError(t, err)
	require.NotNil(t, signup.AccessToken)
	token := *signup.AccessToken

	validator, err := protovalidate.New()
	require.NoError(t, err)

	svc := adminServiceDescriptor(t)
	methods := svc.Methods()

	// AdminLogin is the bootstrap RPC: it exists to establish admin auth, so it
	// is reachable without it. Every other method must deny.
	const bootstrap = "AdminLogin"

	var checkedGRPC, checkedREST int
	var unvalidatable, noHTTPRule []string

	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		name := string(m.Name())
		if name == bootstrap {
			continue
		}

		req := dynamicpb.NewMessage(m.Input())
		fillMessage(req, 0)
		if err := validator.Validate(req); err != nil {
			// The validate interceptor would reject this before the handler, so
			// the authorization gate would never run. Assert nothing.
			unvalidatable = append(unvalidatable, name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			// gRPC: invoked dynamically so the sweep automatically covers RPCs
			// added after this test was written.
			resp := dynamicpb.NewMessage(m.Output())
			err := s.conn.Invoke(s.grpcBearer(token),
				"/"+string(svc.FullName())+"/"+name, req, resp)
			require.Error(t, err, "gRPC served an admin RPC to a plain user")
			assert.Contains(t,
				[]codes.Code{codes.Unauthenticated, codes.PermissionDenied},
				status.Code(err),
				"gRPC denied for the wrong reason (want an authorization refusal, got %v: %v)",
				status.Code(err), err)
			checkedGRPC++

			verb, path, ok := httpRule(m)
			if !ok {
				noHTTPRule = append(noHTTPRule, name)
				return
			}
			body, err := protojson.Marshal(req)
			require.NoError(t, err)
			code := s.restProbe(t, verb, path, token, string(body))
			assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, code,
				"REST %s %s served an admin endpoint to a plain user (status %d)", verb, path, code)
			checkedREST++
		})
	}

	// A sweep that silently checked nothing would pass, so pin the coverage and
	// name everything that was skipped — a growing skip list is a signal, not a
	// detail to bury.
	sort.Strings(unvalidatable)
	t.Logf("denied a plain user on %d admin RPCs over gRPC and %d over REST", checkedGRPC, checkedREST)
	if len(unvalidatable) > 0 {
		t.Logf("skipped %d RPC(s) whose request could not be filled to pass protovalidate "+
			"(validation precedes the gate, so no conclusion is possible): %s",
			len(unvalidatable), strings.Join(unvalidatable, ", "))
	}
	if len(noHTTPRule) > 0 {
		t.Logf("no REST binding (gRPC-only): %s", strings.Join(noHTTPRule, ", "))
	}
	require.Greater(t, checkedGRPC, 60, "admin RPC sweep covered too few methods to be meaningful")
	require.Greater(t, checkedREST, 60, "admin REST sweep covered too few endpoints to be meaningful")
}

// TestEveryRPCHasARoutableRESTBinding checks the REST half of the parity
// promise for both services: every RPC declares a google.api.http rule, and
// every one of those rules actually resolves in the gateway mux.
//
// handlers.TestEveryRPCHasAHandler proves each RPC has a real implementation
// behind it; this proves the generated REST route reaches it. The two together
// are what "every RPC is available over gRPC and REST" means — an RPC can have
// a working handler and still be unreachable over REST if its binding was never
// declared or never mounted.
//
// Payloads are deliberately empty: the assertion is about routing, so it must
// hold regardless of what the handler would do with a real request. 404 means
// the path was never mounted; 501 means the route resolved to no handler.
// Anything else — 200, 400, 401 — means the request was routed and answered.
func TestEveryRPCHasARoutableRESTBinding(t *testing.T) {
	s := newSurfaces(t)

	// Self-check: the whole test rests on an unmounted path being observable as
	// a 404. If the mux ever answered everything, every assertion below would
	// pass while proving nothing.
	require.Equal(t, http.StatusNotFound,
		s.restProbe(t, http.MethodPost, "/v1/definitely_not_an_rpc", "", "{}"),
		"an unmounted path must 404, or this test cannot detect a missing binding")

	for _, svcName := range []string{
		"authorizer.v1.AuthorizerService",
		"authorizer.v1.AuthorizerAdminService",
	} {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(svcName))
		require.NoError(t, err)
		svc, ok := d.(protoreflect.ServiceDescriptor)
		require.True(t, ok)
		methods := svc.Methods()
		require.Greater(t, methods.Len(), 0, "%s: zero methods would pass vacuously", svcName)

		t.Run(svcName, func(t *testing.T) {
			checked := 0
			for i := 0; i < methods.Len(); i++ {
				m := methods.Get(i)
				name := string(m.Name())
				verb, path, ok := httpRule(m)
				if !assert.True(t, ok, "%s: no google.api.http binding, so it has no REST surface", name) {
					continue
				}
				code := s.restProbe(t, verb, path, "", "{}")
				assert.NotEqual(t, http.StatusNotFound, code,
					"%s: REST %s %s is not mounted in the gateway", name, verb, path)
				assert.NotEqual(t, http.StatusNotImplemented, code,
					"%s: REST %s %s routes to no handler", name, verb, path)
				checked++
			}
			t.Logf("%s: %d RPCs routable over REST", svcName, checked)
			require.Greater(t, checked, 0, "%s: nothing was checked", svcName)
		})
	}
}

// adminServiceDescriptor resolves the admin service from the global proto
// registry, so the sweep tracks the proto definition rather than a hand-kept
// list that would silently go stale as RPCs are added.
func adminServiceDescriptor(t *testing.T) protoreflect.ServiceDescriptor {
	t.Helper()
	d, err := protoregistry.GlobalFiles.FindDescriptorByName("authorizer.v1.AuthorizerAdminService")
	require.NoError(t, err)
	svc, ok := d.(protoreflect.ServiceDescriptor)
	require.True(t, ok)
	require.Greater(t, svc.Methods().Len(), 70, "admin service descriptor looks wrong")
	return svc
}

// fillMessage populates every field of a request with a value plausible enough
// to satisfy the common protovalidate constraints (min_len, email, uri, uuid).
// It is deliberately generic: the goal is a request that reaches the
// authorization gate, not a semantically meaningful one — every call it is used
// for is expected to be refused before anything is read or written.
func fillMessage(m protoreflect.Message, depth int) {
	if depth > 3 {
		return
	}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		switch {
		case f.IsMap():
			// Maps are not required by any admin request; leaving them empty
			// keeps the payload minimal.
			continue
		case f.IsList():
			l := m.Mutable(f).List()
			l.Append(scalarFor(f, l.NewElement(), depth))
		default:
			m.Set(f, scalarFor(f, m.NewField(f), depth))
		}
	}
}

// scalarFor produces one value for a field, recursing into nested messages.
func scalarFor(f protoreflect.FieldDescriptor, seed protoreflect.Value, depth int) protoreflect.Value {
	switch f.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(stringFor(string(f.Name())))
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(1)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(1)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(1)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(1)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(1)
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(1)
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte("x"))
	case protoreflect.EnumKind:
		// Index 0 is the UNSPECIFIED convention, which validators usually
		// reject; prefer the first real value when one exists.
		values := f.Enum().Values()
		if values.Len() > 1 {
			return protoreflect.ValueOfEnum(values.Get(1).Number())
		}
		return protoreflect.ValueOfEnum(values.Get(0).Number())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		nested := seed.Message()
		fillMessage(nested, depth+1)
		return protoreflect.ValueOfMessage(nested)
	}
	return seed
}

// stringFor picks a value shaped like what the field name implies, so
// format-constrained fields (email, uri, uuid) validate.
func stringFor(name string) string {
	n := strings.ToLower(name)
	switch {
	// Checked before "email" so an email_domain field gets a domain, not an
	// address. Domain fields are parsed (normalizeDomain) before the gate on
	// the org-domain endpoints, so an unparseable value would stop the probe
	// short of the authorization check it exists to exercise.
	case strings.Contains(n, "domain"):
		return "probe.authorizer.dev"
	case strings.Contains(n, "email"):
		return "probe@authorizer.dev"
	case strings.Contains(n, "url"), strings.Contains(n, "uri"),
		strings.Contains(n, "endpoint"), strings.Contains(n, "issuer"):
		return "https://probe.authorizer.dev"
	case strings.HasSuffix(n, "id"), strings.Contains(n, "_id"):
		return uuid.NewString()
	case strings.Contains(n, "phone"):
		return "+919999999999"
	default:
		return "probe"
	}
}

// httpRule extracts the REST verb and path grpc-gateway generated for an RPC
// from its google.api.http annotation.
func httpRule(m protoreflect.MethodDescriptor) (verb, path string, ok bool) {
	opts := m.Options()
	if opts == nil {
		return "", "", false
	}
	rule, _ := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	if rule == nil {
		return "", "", false
	}
	switch {
	case rule.GetPost() != "":
		verb, path = http.MethodPost, rule.GetPost()
	case rule.GetGet() != "":
		verb, path = http.MethodGet, rule.GetGet()
	case rule.GetPut() != "":
		verb, path = http.MethodPut, rule.GetPut()
	case rule.GetPatch() != "":
		verb, path = http.MethodPatch, rule.GetPatch()
	case rule.GetDelete() != "":
		verb, path = http.MethodDelete, rule.GetDelete()
	default:
		return "", "", false
	}
	return verb, fillPathParams(path), true
}

// fillPathParams replaces {name} / {name=*} segments with a placeholder so the
// gateway routes to the handler instead of 404-ing on the pattern.
func fillPathParams(path string) string {
	var b strings.Builder
	for {
		open := strings.IndexByte(path, '{')
		if open < 0 {
			b.WriteString(path)
			return b.String()
		}
		end := strings.IndexByte(path[open:], '}')
		if end < 0 {
			b.WriteString(path)
			return b.String()
		}
		b.WriteString(path[:open])
		b.WriteString(uuid.NewString())
		path = path[open+end+1:]
	}
}

// restProbe issues a REST call with a bearer token. Only the status code
// matters: the assertion is about the authorization verdict, not the payload.
func (s *surfaces) restProbe(t *testing.T, verb, path, token, body string) int {
	t.Helper()
	if verb == http.MethodGet || verb == http.MethodDelete {
		body = ""
	}
	req, err := http.NewRequestWithContext(context.Background(), verb, s.restURL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authorizer-URL", s.issuer)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}
