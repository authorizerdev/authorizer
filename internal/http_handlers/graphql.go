package http_handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/99designs/gqlgen/complexity"
	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/delegatedscope"
	"github.com/authorizerdev/authorizer/internal/graph"
	"github.com/authorizerdev/authorizer/internal/graph/generated"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// kindToGraphQLCode maps a transport-neutral service.ErrorKind onto a stable
// extensions.code string, mirroring the gRPC code names used by
// grpcsrv/interceptors.ErrorMap so clients only need to learn one vocabulary
// across transports. Only errors explicitly constructed via the service
// package's typed constructors (InvalidArgument, TooManyRequests, ...) reach
// this — every other error is left with no code, exactly preserving today's
// GraphQL behaviour for the vast majority of unclassified/internal errors.
func kindToGraphQLCode(kind service.ErrorKind) string {
	switch kind {
	case service.KindInvalidArgument:
		return "INVALID_ARGUMENT"
	case service.KindUnauthenticated:
		return "UNAUTHENTICATED"
	case service.KindPermissionDenied:
		return "PERMISSION_DENIED"
	case service.KindNotFound:
		return "NOT_FOUND"
	case service.KindFailedPrecondition:
		return "FAILED_PRECONDITION"
	case service.KindTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case service.KindAlreadyExists:
		return "ALREADY_EXISTS"
	default:
		return "INTERNAL"
	}
}

// graphQLErrorPresenter wraps gqlgen's default presenter to attach
// extensions.code for typed service errors, without altering message text or
// touching errors that aren't service.Error (preserving existing behaviour).
func graphQLErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	gqlErr := gql.DefaultErrorPresenter(ctx, e)
	var svcErr *service.Error
	if errors.As(e, &svcErr) {
		if gqlErr.Extensions == nil {
			gqlErr.Extensions = map[string]interface{}{}
		}
		gqlErr.Extensions["code"] = kindToGraphQLCode(svcErr.Kind)
	}
	return gqlErr
}

// queryLimits is a gqlgen handler extension that enforces depth, alias, and
// complexity limits on parsed operations. It runs after parsing but before
// execution so abusive queries are rejected without consuming resolver work.
//
// We replace gqlgen's stock extension.FixedComplexityLimit so all three
// limits go through the same code path and emit the same Prometheus
// counter (authorizer_graphql_limit_rejections_total) labelled by the
// specific limit kind. Operators can then alert on a sustained non-zero
// rate per limit and tune individually.
type queryLimits struct {
	maxDepth      int
	maxAliases    int
	maxComplexity int
	schema        gql.ExecutableSchema
}

var (
	_ gql.HandlerExtension        = (*queryLimits)(nil)
	_ gql.OperationContextMutator = (*queryLimits)(nil)
)

func (*queryLimits) ExtensionName() string { return "QueryLimits" }
func (q *queryLimits) Validate(schema gql.ExecutableSchema) error {
	q.schema = schema
	return nil
}
func (q *queryLimits) MutateOperationContext(ctx context.Context, rc *gql.OperationContext) *gqlerror.Error {
	if rc == nil || rc.Operation == nil {
		return nil
	}
	// Single AST walk computes both max depth and total alias count so we
	// touch each selection-set node exactly once. The earlier two-pass
	// implementation walked the same tree twice for legitimate traffic;
	// folding them halves the per-request AST work.
	if q.maxDepth > 0 || q.maxAliases > 0 {
		depth, aliases := walkSelectionSet(rc.Operation.SelectionSet)
		if q.maxDepth > 0 && depth > q.maxDepth {
			metrics.RecordGraphQLLimitRejection(metrics.GraphQLLimitDepth)
			return gqlerror.Errorf("query depth %d exceeds maximum allowed depth %d", depth, q.maxDepth)
		}
		if q.maxAliases > 0 && aliases > q.maxAliases {
			metrics.RecordGraphQLLimitRejection(metrics.GraphQLLimitAlias)
			return gqlerror.Errorf("query uses %d aliases, exceeds maximum %d", aliases, q.maxAliases)
		}
	}
	if q.maxComplexity > 0 && q.schema != nil {
		score := complexity.Calculate(ctx, q.schema, rc.Operation, rc.Variables)
		if score > q.maxComplexity {
			metrics.RecordGraphQLLimitRejection(metrics.GraphQLLimitComplexity)
			return gqlerror.Errorf("operation has complexity %d, which exceeds the limit of %d", score, q.maxComplexity)
		}
	}
	return nil
}

// walkSelectionSet returns (max nesting depth, total alias count) for the
// supplied selection set in a single recursive pass. Inline fragments and
// fragment spreads do not contribute their own depth level (matching the
// usual GraphQL convention) but their aliases do count.
func walkSelectionSet(set ast.SelectionSet) (depth, aliases int) {
	for _, sel := range set {
		switch s := sel.(type) {
		case *ast.Field:
			if s.Alias != "" && s.Alias != s.Name {
				aliases++
			}
			childDepth, childAliases := walkSelectionSet(s.SelectionSet)
			aliases += childAliases
			if d := 1 + childDepth; d > depth {
				depth = d
			}
		case *ast.InlineFragment:
			childDepth, childAliases := walkSelectionSet(s.SelectionSet)
			aliases += childAliases
			if childDepth > depth {
				depth = childDepth
			}
		case *ast.FragmentSpread:
			if s.Definition != nil {
				childDepth, childAliases := walkSelectionSet(s.Definition.SelectionSet)
				aliases += childAliases
				if childDepth > depth {
					depth = childDepth
				}
			}
		}
	}
	return depth, aliases
}

type gqlResolvedFieldsCtxKey struct{}

// resolvedFieldsCollector gathers unique GraphQL field names for one operation.
type resolvedFieldsCollector struct {
	mu     sync.Mutex
	fields map[string]struct{}
}

func (c *resolvedFieldsCollector) add(name string) {
	if name == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fields == nil {
		c.fields = make(map[string]struct{})
	}
	c.fields[name] = struct{}{}
}

func (c *resolvedFieldsCollector) sortedUnique() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.fields))
	for f := range c.fields {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// gqlCollectResolvedFieldsMiddleware records each resolved field name into the per-operation collector.
func (*httpProvider) gqlCollectResolvedFieldsMiddleware() gql.FieldMiddleware {
	return func(ctx context.Context, next gql.Resolver) (interface{}, error) {
		if col, ok := ctx.Value(gqlResolvedFieldsCtxKey{}).(*resolvedFieldsCollector); ok && col != nil {
			if fc := gql.GetFieldContext(ctx); fc != nil && fc.Field.Field != nil {
				col.add(fc.Field.Name)
			}
		}
		return next(ctx)
	}
}

// gqlDelegatedScopeMiddleware gates a DELEGATED caller — an RFC 8693 agent
// acting for a user — on the scope its token actually carries.
//
// GraphQL is the other transport that needs this; gRPC (and with it the REST
// gateway and the MCP server) is handled in grpcsrv/interceptors. Both consult
// the same table in internal/delegatedscope, so the two cannot disagree about
// what an agent may do.
//
// Applied at ROOT fields only. Nested field resolvers inherit the decision of
// the operation that reached them, and re-checking every leaf would both cost
// a map lookup per field and, worse, deny nested selections that have no entry
// of their own.
//
// First-party callers are not gated here. See the delegatedscope package
// comment: a first-party `scope` is caller-supplied and unvalidated, so it is a
// hint rather than a boundary, and enforcing it would break existing clients
// for no security gain.
func (h *httpProvider) gqlDelegatedScopeMiddleware() gql.FieldMiddleware {
	return func(ctx context.Context, next gql.Resolver) (interface{}, error) {
		fc := gql.GetFieldContext(ctx)
		// Root fields only. Deliberately NOT also gated on fc.IsMethod: that
		// reports how gqlgen resolves the field, which is an implementation
		// detail of the generated code, and a false there would skip the gate
		// entirely. Object is the schema-level fact this actually depends on.
		if fc == nil || fc.Field.Field == nil || (fc.Object != "Query" && fc.Object != "Mutation") {
			return next(ctx)
		}
		gc, err := utils.GinContextFromContext(ctx)
		if err != nil || gc == nil {
			return next(ctx)
		}

		// Decided from the JWT alone — signature and claims, no storage.
		//
		// The first version resolved the caller through
		// GetUserIDFromSessionOrAccessToken and skipped the gate whenever that
		// returned an error. For a delegated token that call reads the session
		// store and the user row, so a transient storage failure between this
		// check and the resolver's own auth would have skipped the gate while
		// the request still succeeded: fail-OPEN on the one path that exists to
		// fail closed. Whether a token is delegated, and what it is scoped to,
		// are properties of the signed token itself; nothing about that needs
		// a database.
		//
		// It is also ~free, which matters because this runs per root field.
		raw, tErr := h.TokenProvider.GetAccessToken(gc)
		if tErr != nil || strings.TrimSpace(raw) == "" {
			// No bearer token at all — a cookie/session or anonymous caller.
			// Neither can be delegated; the resolver's own auth owns them.
			return next(ctx)
		}
		claims, cErr := h.TokenProvider.ParseJWTToken(raw)
		if cErr != nil {
			// Unparseable or badly signed. Not our decision to make — the
			// resolver's auth will reject it.
			return next(ctx)
		}
		if token.ImmediateActor(claims) == "" {
			return next(ctx) // first-party token; see the package comment.
		}

		// Both refusals below are metered. This is the agent scope-ceiling
		// enforcement point, and it was previously silent on every transport: an
		// operator had no way to see that agents were hitting their ceiling, which
		// is exactly the number needed before deciding whether to widen one. The
		// two outcomes are separate labels because they call for opposite actions
		// — "not_delegatable" means the operation is not on the delegated
		// allow-list at all (a client bug, or probing), while "scope_missing"
		// means it is reachable but this token was not granted it.
		required, ok := delegatedscope.RequiredForGraphQL(fc.Field.Name)
		if !ok {
			// Fail closed: an operation nobody has cleared for delegated
			// callers is out of reach for an agent, whatever scope it holds.
			metrics.RecordSecurityEvent("delegated_insufficient_scope", "graphql_not_delegatable")
			return nil, gqlerror.Errorf("insufficient_scope")
		}
		if !delegatedscope.Satisfied(token.ClaimScopes(claims), required) {
			metrics.RecordSecurityEvent("delegated_insufficient_scope", "graphql_scope_missing")
			return nil, gqlerror.Errorf("insufficient_scope")
		}
		return next(ctx)
	}
}

// gqlMetricsMiddleware records GraphQL operation duration and errors.
// It captures errors returned in HTTP 200 responses (GraphQL convention).
func (h *httpProvider) gqlMetricsMiddleware() gql.OperationMiddleware {
	return func(ctx context.Context, next gql.OperationHandler) gql.ResponseHandler {
		operationName := ""
		if oc := gql.GetOperationContext(ctx); oc != nil {
			operationName = oc.OperationName
		}
		opMetricLabel := metrics.GraphQLOperationPrometheusLabel(operationName)
		start := time.Now()

		collector := &resolvedFieldsCollector{}
		ctx = context.WithValue(ctx, gqlResolvedFieldsCtxKey{}, collector)

		responseHandler := next(ctx)

		return func(ctx context.Context) *gql.Response {
			resp := responseHandler(ctx)
			fields := collector.sortedUnique()
			if resp == nil {
				h.Dependencies.Log.Warn().
					Str("operation", operationName).
					Str("operation_metric_label", opMetricLabel).
					Strs("resolved_fields", fields).
					Msg("GraphQL operation returned no response")
				return resp
			}
			duration := time.Since(start).Seconds()
			metrics.GraphQLRequestDuration.WithLabelValues(opMetricLabel).Observe(duration)

			opStatus := metrics.OperationStatusOK
			if len(resp.Errors) > 0 {
				metrics.RecordGraphQLError(operationName)
				opStatus = metrics.OperationStatusError
			}
			// Attribute the operation to the GraphQL protocol for the shared
			// api-operations metric (gRPC/REST are recorded in the gRPC
			// interceptor). opMetricLabel is the low-cardinality op label.
			metrics.RecordAPIOperation(constants.ProtocolGraphQL, opMetricLabel, opStatus)
			logEvt := h.Dependencies.Log.Info().
				Str("operation", operationName).
				Str("operation_metric_label", opMetricLabel).
				Int("resolved_field_count", len(fields))
			logEvt.Msg("GraphQL operation completed")
			h.Dependencies.Log.Debug().
				Str("operation", operationName).
				Strs("resolved_fields", fields).
				Msg("GraphQL resolved fields")
			return resp
		}
	}
}

// GraphqlHandler is the main handler that handles all GraphQL requests.
func (h *httpProvider) GraphqlHandler() gin.HandlerFunc {
	gqlProvider, err := graphql.New(h.Config, &graphql.Dependencies{
		Log:                   h.Log,
		AuditProvider:         h.AuditProvider,
		AuthenticatorProvider: h.AuthenticatorProvider,
		EmailProvider:         h.EmailProvider,
		EventsProvider:        h.EventsProvider,
		MemoryStoreProvider:   h.MemoryStoreProvider,
		SMSProvider:           h.SMSProvider,
		StorageProvider:       h.StorageProvider,
		TokenProvider:         h.TokenProvider,
		ServiceProvider:       h.ServiceProvider,
		AuthzEngine:           h.AuthzEngine,
	})
	if err != nil {
		h.Log.Error().Err(err).Msg("Failed to create graphql provider")
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "graphql_unavailable",
				"error_description": "GraphQL service failed to initialize.",
			})
		}
	}

	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		GraphQLProvider: gqlProvider,
	}}))

	srv.SetErrorPresenter(graphQLErrorPresenter)

	srv.AddTransport(transport.Options{})
	// transport.GET is intentionally omitted: GraphQL queries (and especially
	// mutations) over GET leak into proxy/server logs and browser history.
	// Clients must POST.
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.AroundFields(h.gqlCollectResolvedFieldsMiddleware())
	srv.AroundFields(h.gqlDelegatedScopeMiddleware())
	srv.AroundOperations(h.gqlMetricsMiddleware())
	if h.Config.EnableGraphQLIntrospection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Limit query depth, alias count, AND complexity through a single
	// extension so all three rejections share one Prometheus counter
	// (authorizer_graphql_limit_rejections_total). Defaults applied if
	// config is unset.
	maxComplexity := h.Config.GraphQLMaxComplexity
	if maxComplexity <= 0 {
		maxComplexity = 300
	}
	maxDepth := h.Config.GraphQLMaxDepth
	if maxDepth <= 0 {
		maxDepth = 15
	}
	maxAliases := h.Config.GraphQLMaxAliases
	if maxAliases <= 0 {
		maxAliases = 30
	}
	srv.Use(&queryLimits{
		maxDepth:      maxDepth,
		maxAliases:    maxAliases,
		maxComplexity: maxComplexity,
	})

	// Cap the request body size to defend against oversized-payload DoS.
	maxBody := h.Config.GraphQLMaxBodyBytes
	if maxBody <= 0 {
		maxBody = 1 << 20 // 1 MB
	}

	return func(c *gin.Context) {
		// Create a custom handler that ensures gin context is available
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bound the request body so a single client cannot exhaust memory.
			// http.MaxBytesReader will return an error from r.Body.Read once
			// the limit is exceeded; gqlgen surfaces that as a parse error.
			// We wrap the writer in a sniffer so we can detect the error and
			// emit the body_size limit metric.
			r.Body = &maxBytesBody{
				ReadCloser: http.MaxBytesReader(w, r.Body, maxBody),
			}
			// Ensure the gin context is available in the request context
			ctx := utils.ContextWithGin(r.Context(), c)
			r = r.WithContext(ctx)
			srv.ServeHTTP(w, r)
			// If the body reader hit the cap, record the rejection. We do
			// this once per request after the handler returns so the metric
			// reflects actual aborts, not just oversized-but-streaming reads.
			if mb, ok := r.Body.(*maxBytesBody); ok && mb.exceeded {
				metrics.RecordGraphQLLimitRejection(metrics.GraphQLLimitBodySize)
			}
		})
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// maxBytesBody wraps the io.ReadCloser returned by http.MaxBytesReader so
// the request handler can tell after the fact whether the body exceeded
// the configured cap. http.MaxBytesReader signals exhaustion via a
// *http.MaxBytesError wrapping io.EOF, but the gqlgen handler swallows the
// error inside its parse step — we need to observe the read directly to
// emit the body_size limit rejection metric.
type maxBytesBody struct {
	io.ReadCloser
	exceeded bool
}

func (m *maxBytesBody) Read(p []byte) (int, error) {
	n, err := m.ReadCloser.Read(p)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			m.exceeded = true
		}
	}
	return n, err
}
