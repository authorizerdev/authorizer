package handlers

import (
	"context"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// Every RPC declared in the proto must have a real handler.
//
// Both handler structs embed the generated Unimplemented…ServiceServer, which
// is what lets the package compile while RPCs are added incrementally. The cost
// is that FORGETTING a handler is invisible: the build succeeds and the RPC
// fails at runtime with codes.Unimplemented. Nothing else in the suite catches
// that.
//
// Detection is behavioural, not name-based. Go synthesises a wrapper method on
// the outer type for a promoted method, so reflection reports the identical
// name ("(*AuthorizerHandler).Foo") whether Foo is declared here or inherited
// from the embedded fallback — a name check silently passes for everything.
// Instead each method is invoked on a handler with a NIL Service:
//
//   - the Unimplemented fallback returns codes.Unimplemented without touching
//     Service            => the RPC has no handler, fail.
//   - a real handler dereferences the nil Service and panics, or returns some
//     other error         => the RPC is implemented, pass.
func TestEveryRPCHasAHandler(t *testing.T) {
	for _, tc := range []struct {
		name    string
		desc    grpc.ServiceDesc
		handler any
	}{
		{"AuthorizerService", authorizerv1.AuthorizerService_ServiceDesc, &AuthorizerHandler{}},
		{"AuthorizerAdminService", authorizerv1.AuthorizerAdminService_ServiceDesc, &AdminHandler{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.desc.Methods) == 0 {
				t.Fatalf("%s: descriptor reported zero methods — the check would vacuously pass", tc.name)
			}
			hv := reflect.ValueOf(tc.handler)
			ht := reflect.TypeOf(tc.handler)
			for _, m := range tc.desc.Methods {
				method := hv.MethodByName(m.MethodName)
				if !method.IsValid() {
					t.Errorf("%s.%s: no method on the handler at all", tc.name, m.MethodName)
					continue
				}
				mt, _ := ht.MethodByName(m.MethodName)
				if mt.Type.NumIn() != 3 { // receiver, ctx, request
					t.Errorf("%s.%s: unexpected signature %s", tc.name, m.MethodName, mt.Type)
					continue
				}
				if unimplemented(method, mt.Type.In(2)) {
					t.Errorf("%s.%s: returns codes.Unimplemented — declared in proto but has no "+
						"handler, so it fails at runtime for every caller", tc.name, m.MethodName)
				}
			}
			t.Logf("%s: %d RPCs, all backed by a real handler", tc.name, len(tc.desc.Methods))
		})
	}
}

// unimplemented reports whether calling method with a zero request yields a
// codes.Unimplemented status. A panic means real code ran against the nil
// Service, which is exactly the signal we want.
func unimplemented(method reflect.Value, reqType reflect.Type) (isFallback bool) {
	defer func() {
		if recover() != nil {
			isFallback = false // real handler: it dereferenced the nil Service
		}
	}()
	req := reflect.New(reqType.Elem())
	out := method.Call([]reflect.Value{reflect.ValueOf(context.Background()), req})
	errVal := out[len(out)-1]
	if errVal.IsNil() {
		return false
	}
	return status.Code(errVal.Interface().(error)) == codes.Unimplemented
}
