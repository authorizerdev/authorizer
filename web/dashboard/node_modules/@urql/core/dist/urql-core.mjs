import { Kind as e } from "@0no-co/graphql.web";

import { k as r, s as t, m as n, C as a, a as o, b as s, c, d as u, e as p, f as d, g as l, h as v } from "./urql-core-chunk.mjs";

export { i as stringifyVariables } from "./urql-core-chunk.mjs";

import { toPromise as f, take as h, filter as k, subscribe as y, map as m, tap as x, merge as g, mergeMap as w, takeUntil as O, make as E, onPush as b, share as N, fromPromise as q, fromValue as _, makeSubject as D, lazy as S, onStart as P, switchMap as R, publish as V, takeWhile as A, onEnd as M } from "wonka";

var collectTypes = (e, r) => {
  if (Array.isArray(e)) {
    for (var t = 0, n = e.length; t < n; t++) {
      collectTypes(e[t], r);
    }
  } else if ("object" == typeof e && null !== e) {
    for (var a in e) {
      if ("__typename" === a && "string" == typeof e[a]) {
        r.add(e[a]);
      } else {
        collectTypes(e[a], r);
      }
    }
  }
  return r;
};

var formatNode = r => {
  if ("definitions" in r) {
    var t = [];
    for (var n = 0, a = r.definitions.length; n < a; n++) {
      var i = formatNode(r.definitions[n]);
      t.push(i);
    }
    return {
      ...r,
      definitions: t
    };
  }
  if ("directives" in r && r.directives && r.directives.length) {
    var o = [];
    var s = {};
    for (var c = 0, u = r.directives.length; c < u; c++) {
      var p = r.directives[c];
      var d = p.name.value;
      if ("_" !== d[0]) {
        o.push(p);
      } else {
        d = d.slice(1);
      }
      s[d] = p;
    }
    r = {
      ...r,
      directives: o,
      _directives: s
    };
  }
  if ("selectionSet" in r) {
    var l = [];
    var v = r.kind === e.OPERATION_DEFINITION;
    if (r.selectionSet) {
      for (var f = 0, h = r.selectionSet.selections.length; f < h; f++) {
        var k = r.selectionSet.selections[f];
        v = v || k.kind === e.FIELD && "__typename" === k.name.value && !k.alias;
        var y = formatNode(k);
        l.push(y);
      }
      if (!v) {
        l.push({
          kind: e.FIELD,
          name: {
            kind: e.NAME,
            value: "__typename"
          },
          _generated: !0
        });
      }
      return {
        ...r,
        selectionSet: {
          ...r.selectionSet,
          selections: l
        }
      };
    }
  }
  return r;
};

var I = new Map;

var formatDocument = e => {
  var t = r(e);
  var n = I.get(t.__key);
  if (!n) {
    I.set(t.__key, n = formatNode(t));
    Object.defineProperty(n, "__key", {
      value: t.__key,
      enumerable: !1
    });
  }
  return n;
};

function withPromise(e) {
  var source$ = r => e(r);
  source$.toPromise = () => f(h(1)(k((e => !e.stale && !e.hasNext))(source$)));
  source$.then = (e, r) => source$.toPromise().then(e, r);
  source$.subscribe = e => y(e)(source$);
  return source$;
}

function makeOperation(e, r, t) {
  return {
    ...r,
    kind: e,
    context: r.context ? {
      ...r.context,
      ...t
    } : t || r.context
  };
}

var addMetadata = (e, r) => makeOperation(e.kind, e, {
  meta: {
    ...e.context.meta,
    ...r
  }
});

var noop = () => {};

function gql(n) {
  var a = new Map;
  var i = [];
  var o = [];
  var s = Array.isArray(n) ? n[0] : n || "";
  for (var c = 1; c < arguments.length; c++) {
    var u = arguments[c];
    if (u && u.definitions) {
      o.push(u);
    } else {
      s += u;
    }
    s += arguments[0][c];
  }
  o.unshift(r(s));
  for (var p = 0; p < o.length; p++) {
    for (var d = 0; d < o[p].definitions.length; d++) {
      var l = o[p].definitions[d];
      if (l.kind === e.FRAGMENT_DEFINITION) {
        var v = l.name.value;
        var f = t(l);
        if (!a.has(v)) {
          a.set(v, f);
          i.push(l);
        } else if ("production" !== process.env.NODE_ENV && a.get(v) !== f) {
          console.warn("[WARNING: Duplicate Fragment] A fragment with name `" + v + "` already exists in this document.\nWhile fragment names may not be unique across your source, each name must be unique per document.");
        }
      } else {
        i.push(l);
      }
    }
  }
  return r({
    kind: e.DOCUMENT,
    definitions: i
  });
}

var shouldSkip = ({kind: e}) => "mutation" !== e && "query" !== e;

var mapTypeNames = e => {
  var r = formatDocument(e.query);
  if (r !== e.query) {
    var t = makeOperation(e.kind, e);
    t.query = r;
    return t;
  } else {
    return e;
  }
};

var cacheExchange = ({forward: e, client: r, dispatchDebug: t}) => {
  var a = new Map;
  var i = new Map;
  var isOperationCached = e => "query" === e.kind && "network-only" !== e.context.requestPolicy && ("cache-only" === e.context.requestPolicy || a.has(e.key));
  return o => {
    var s = m((e => {
      var i = a.get(e.key);
      "production" !== process.env.NODE_ENV && t({
        operation: e,
        ...i ? {
          type: "cacheHit",
          message: "The result was successfully retried from the cache"
        } : {
          type: "cacheMiss",
          message: "The result could not be retrieved from the cache"
        },
        source: "cacheExchange"
      });
      var o = i || n(e, {
        data: null
      });
      o = {
        ...o,
        operation: addMetadata(e, {
          cacheOutcome: i ? "hit" : "miss"
        })
      };
      if ("cache-and-network" === e.context.requestPolicy) {
        o.stale = !0;
        reexecuteOperation(r, e);
      }
      return o;
    }))(k((e => !shouldSkip(e) && isOperationCached(e)))(o));
    var c = x((e => {
      var {operation: n} = e;
      if (!n) {
        return;
      }
      var o = n.context.additionalTypenames || [];
      if ("subscription" !== e.operation.kind) {
        o = (e => [ ...collectTypes(e, new Set) ])(e.data).concat(o);
      }
      if ("mutation" === e.operation.kind || "subscription" === e.operation.kind) {
        var s = new Set;
        "production" !== process.env.NODE_ENV && t({
          type: "cacheInvalidation",
          message: `The following typenames have been invalidated: ${o}`,
          operation: n,
          data: {
            typenames: o,
            response: e
          },
          source: "cacheExchange"
        });
        for (var c = 0; c < o.length; c++) {
          var u = o[c];
          var p = i.get(u);
          if (!p) {
            i.set(u, p = new Set);
          }
          for (var d of p.values()) {
            s.add(d);
          }
          p.clear();
        }
        for (var l of s.values()) {
          if (a.has(l)) {
            n = a.get(l).operation;
            a.delete(l);
            reexecuteOperation(r, n);
          }
        }
      } else if ("query" === n.kind && e.data) {
        a.set(n.key, e);
        for (var v = 0; v < o.length; v++) {
          var f = o[v];
          var h = i.get(f);
          if (!h) {
            i.set(f, h = new Set);
          }
          h.add(n.key);
        }
      }
    }))(e(k((e => "query" !== e.kind || "cache-only" !== e.context.requestPolicy))(m((e => addMetadata(e, {
      cacheOutcome: "miss"
    })))(g([ m(mapTypeNames)(k((e => !shouldSkip(e) && !isOperationCached(e)))(o)), k((e => shouldSkip(e)))(o) ])))));
    return g([ s, c ]);
  };
};

var reexecuteOperation = (e, r) => e.reexecuteOperation(makeOperation(r.kind, r, {
  requestPolicy: "network-only"
}));

var T = new Set;

var ssrExchange = (e = {}) => {
  var r = !!e.staleWhileRevalidate;
  var t = !!e.includeExtensions;
  var n = {};
  var i = [];
  var invalidate = e => {
    i.push(e.operation.key);
    if (1 === i.length) {
      Promise.resolve().then((() => {
        var e;
        while (e = i.shift()) {
          n[e] = null;
        }
      }));
    }
  };
  var ssr = ({client: i, forward: o}) => s => {
    var c = e && "boolean" == typeof e.isClient ? !!e.isClient : !i.suspense;
    var u = o(m(mapTypeNames)(k((e => "teardown" === e.kind || !n[e.key] || !!n[e.key].hasNext || "network-only" === e.context.requestPolicy))(s)));
    var p = m((e => {
      var o = ((e, r, t) => ({
        operation: e,
        data: r.data ? JSON.parse(r.data) : void 0,
        extensions: t && r.extensions ? JSON.parse(r.extensions) : void 0,
        error: r.error ? new a({
          networkError: r.error.networkError ? new Error(r.error.networkError) : void 0,
          graphQLErrors: r.error.graphQLErrors
        }) : void 0,
        stale: !1,
        hasNext: !!r.hasNext
      }))(e, n[e.key], t);
      if (r && !T.has(e.key)) {
        o.stale = !0;
        T.add(e.key);
        reexecuteOperation(i, e);
      }
      return {
        ...o,
        operation: addMetadata(e, {
          cacheOutcome: "hit"
        })
      };
    }))(k((e => "teardown" !== e.kind && !!n[e.key] && "network-only" !== e.context.requestPolicy))(s));
    if (!c) {
      u = x((e => {
        var {operation: r} = e;
        if ("mutation" !== r.kind) {
          var a = ((e, r) => {
            var t = {
              hasNext: e.hasNext
            };
            if (void 0 !== e.data) {
              t.data = JSON.stringify(e.data);
            }
            if (r && void 0 !== e.extensions) {
              t.extensions = JSON.stringify(e.extensions);
            }
            if (e.error) {
              t.error = {
                graphQLErrors: e.error.graphQLErrors.map((e => {
                  if (!e.path && !e.extensions) {
                    return e.message;
                  }
                  return {
                    message: e.message,
                    path: e.path,
                    extensions: e.extensions
                  };
                }))
              };
              if (e.error.networkError) {
                t.error.networkError = "" + e.error.networkError;
              }
            }
            return t;
          })(e, t);
          n[r.key] = a;
        }
      }))(u);
    } else {
      p = x(invalidate)(p);
    }
    return g([ u, p ]);
  };
  ssr.restoreData = e => {
    for (var r in e) {
      if (null !== n[r]) {
        n[r] = e[r];
      }
    }
  };
  ssr.extractData = () => {
    var e = {};
    for (var r in n) {
      if (null != n[r]) {
        e[r] = n[r];
      }
    }
    return e;
  };
  if (e && e.initialState) {
    ssr.restoreData(e.initialState);
  }
  return ssr;
};

var subscriptionExchange = ({forwardSubscription: e, enableAllOperations: r, isSubscriptionOperation: t}) => ({client: a, forward: i}) => {
  var u = t || (e => "subscription" === e.kind || !!r && ("query" === e.kind || "mutation" === e.kind));
  return r => {
    var t = w((t => {
      var {key: i} = t;
      var u = k((e => "teardown" === e.kind && e.key === i))(r);
      return O(u)((r => {
        var t = e(o(r), r);
        return E((e => {
          var i = !1;
          var o;
          var u;
          function nextResult(t) {
            e.next(u = u ? c(u, t) : n(r, t));
          }
          Promise.resolve().then((() => {
            if (i) {
              return;
            }
            o = t.subscribe({
              next: nextResult,
              error(t) {
                if (Array.isArray(t)) {
                  nextResult({
                    errors: t
                  });
                } else {
                  e.next(s(r, t));
                }
                e.complete();
              },
              complete() {
                if (!i) {
                  i = !0;
                  if ("subscription" === r.kind) {
                    a.reexecuteOperation(makeOperation("teardown", r, r.context));
                  }
                  if (u && u.hasNext) {
                    nextResult({
                      hasNext: !1
                    });
                  }
                  e.complete();
                }
              }
            });
          }));
          return () => {
            i = !0;
            if (o) {
              o.unsubscribe();
            }
          };
        }));
      })(t));
    }))(k((e => "teardown" !== e.kind && u(e)))(r));
    var p = i(k((e => "teardown" === e.kind || !u(e)))(r));
    return g([ t, p ]);
  };
};

var debugExchange = ({forward: e}) => {
  if ("production" === process.env.NODE_ENV) {
    return r => e(r);
  } else {
    return r => x((e => console.log("[Exchange debug]: Completed operation: ", e)))(e(x((e => console.log("[Exchange debug]: Incoming operation: ", e)))(r)));
  }
};

var fetchExchange = ({forward: e, dispatchDebug: r}) => t => {
  var n = w((e => {
    var n = o(e);
    var a = u(e, n);
    var i = p(e, n);
    "production" !== process.env.NODE_ENV && r({
      type: "fetchRequest",
      message: "A fetch request is being executed.",
      operation: e,
      data: {
        url: a,
        fetchOptions: i
      },
      source: "fetchExchange"
    });
    var s = O(k((r => "teardown" === r.kind && r.key === e.key))(t))(d(e, a, i));
    if ("production" !== process.env.NODE_ENV) {
      return b((t => {
        var n = !t.data ? t.error : void 0;
        "production" !== process.env.NODE_ENV && r({
          type: n ? "fetchError" : "fetchSuccess",
          message: `A ${n ? "failed" : "successful"} fetch response has been returned.`,
          operation: e,
          data: {
            url: a,
            fetchOptions: i,
            value: n || t
          },
          source: "fetchExchange"
        });
      }))(s);
    }
    return s;
  }))(k((e => "teardown" !== e.kind && ("subscription" !== e.kind || !!e.context.fetchSubscriptions)))(t));
  var a = e(k((e => "teardown" === e.kind || "subscription" === e.kind && !e.context.fetchSubscriptions))(t));
  return g([ n, a ]);
};

var composeExchanges = e => ({client: r, forward: t, dispatchDebug: n}) => e.reduceRight(((e, t) => {
  var a = !1;
  return t({
    client: r,
    forward(r) {
      if ("production" !== process.env.NODE_ENV) {
        if (a) {
          throw new Error("forward() must only be called once in each Exchange.");
        }
        a = !0;
      }
      return N(e(N(r)));
    },
    dispatchDebug(e) {
      "production" !== process.env.NODE_ENV && n({
        timestamp: Date.now(),
        source: t.name,
        ...e
      });
    }
  });
}), t);

var mapExchange = ({onOperation: e, onResult: r, onError: t}) => ({forward: n}) => a => w((e => {
  if (t && e.error) {
    t(e.error, e.operation);
  }
  var n = r && r(e) || e;
  return "then" in n ? q(n) : _(n);
}))(n(w((r => {
  var t = e && e(r) || r;
  return "then" in t ? q(t) : _(t);
}))(a)));

var fallbackExchange = ({dispatchDebug: e}) => r => {
  if ("production" !== process.env.NODE_ENV) {
    r = x((r => {
      if ("teardown" !== r.kind && "production" !== process.env.NODE_ENV) {
        var t = `No exchange has handled operations of kind "${r.kind}". Check whether you've added an exchange responsible for these operations.`;
        "production" !== process.env.NODE_ENV && e({
          type: "fallbackCatch",
          message: t,
          operation: r,
          source: "fallbackExchange"
        });
        console.warn(t);
      }
    }))(r);
  }
  return k((e => !1))(r);
};

var C = function Client(e) {
  if ("production" !== process.env.NODE_ENV && !e.url) {
    throw new Error("You are creating an urql-client without a url.");
  }
  var r = 0;
  var t = new Map;
  var n = new Map;
  var a = new Set;
  var i = [];
  var o = {
    url: e.url,
    fetchSubscriptions: e.fetchSubscriptions,
    fetchOptions: e.fetchOptions,
    fetch: e.fetch,
    preferGetMethod: e.preferGetMethod,
    requestPolicy: e.requestPolicy || "cache-first"
  };
  var s = D();
  function nextOperation(e) {
    if ("mutation" === e.kind || "teardown" === e.kind || !a.has(e.key)) {
      if ("teardown" === e.kind) {
        a.delete(e.key);
      } else if ("mutation" !== e.kind) {
        a.add(e.key);
      }
      s.next(e);
    }
  }
  var c = !1;
  function dispatchOperation(e) {
    if (e) {
      nextOperation(e);
    }
    if (!c) {
      c = !0;
      while (c && (e = i.shift())) {
        nextOperation(e);
      }
      c = !1;
    }
  }
  var makeResultSource = e => {
    var r = O(k((r => "teardown" === r.kind && r.key === e.key))(s.source))(k((r => r.operation.kind === e.kind && r.operation.key === e.key && (!r.operation.context._instance || r.operation.context._instance === e.context._instance)))(E));
    if ("query" !== e.kind) {
      r = A((e => !!e.hasNext), !0)(r);
    } else {
      r = R((r => {
        var t = _(r);
        return r.stale || r.hasNext ? t : g([ t, m((() => {
          r.stale = !0;
          return r;
        }))(h(1)(k((r => r.key === e.key))(s.source))) ]);
      }))(r);
    }
    if ("mutation" !== e.kind) {
      r = M((() => {
        a.delete(e.key);
        t.delete(e.key);
        n.delete(e.key);
        c = !1;
        for (var r = i.length - 1; r >= 0; r--) {
          if (i[r].key === e.key) {
            i.splice(r, 1);
          }
        }
        nextOperation(makeOperation("teardown", e, e.context));
      }))(b((r => {
        if (r.stale) {
          if (!r.hasNext) {
            a.delete(e.key);
          } else {
            for (var n = 0; n < i.length; n++) {
              var o = i[n];
              if (o.key === r.operation.key) {
                a.delete(o.key);
                break;
              }
            }
          }
        } else if (!r.hasNext) {
          a.delete(e.key);
        }
        t.set(e.key, r);
      }))(r));
    } else {
      r = P((() => {
        nextOperation(e);
      }))(r);
    }
    return N(r);
  };
  var u = this instanceof Client ? this : Object.create(Client.prototype);
  var p = Object.assign(u, {
    suspense: !!e.suspense,
    operations$: s.source,
    reexecuteOperation(e) {
      if ("teardown" === e.kind) {
        dispatchOperation(e);
      } else if ("mutation" === e.kind) {
        i.push(e);
        Promise.resolve().then(dispatchOperation);
      } else if (n.has(e.key)) {
        var r = !1;
        for (var t = 0; t < i.length; t++) {
          if (i[t].key === e.key) {
            i[t] = e;
            r = !0;
          }
        }
        if (!(r || a.has(e.key) && "network-only" !== e.context.requestPolicy)) {
          i.push(e);
          Promise.resolve().then(dispatchOperation);
        } else {
          a.delete(e.key);
          Promise.resolve().then(dispatchOperation);
        }
      }
    },
    createRequestOperation(e, t, n) {
      if (!n) {
        n = {};
      }
      var a;
      if ("production" !== process.env.NODE_ENV && "teardown" !== e && (a = l(t.query)) !== e) {
        throw new Error(`Expected operation of type "${e}" but found "${a}"`);
      }
      return makeOperation(e, t, {
        _instance: "mutation" === e ? r = r + 1 | 0 : void 0,
        ...o,
        ...n,
        requestPolicy: n.requestPolicy || o.requestPolicy,
        suspense: n.suspense || !1 !== n.suspense && p.suspense
      });
    },
    executeRequestOperation(e) {
      if ("mutation" === e.kind) {
        return withPromise(makeResultSource(e));
      }
      return withPromise(S((() => {
        var r = n.get(e.key);
        if (!r) {
          n.set(e.key, r = makeResultSource(e));
        }
        r = P((() => {
          dispatchOperation(e);
        }))(r);
        var a = t.get(e.key);
        if ("query" === e.kind && a && (a.stale || a.hasNext)) {
          return R(_)(g([ r, k((r => r === t.get(e.key)))(_(a)) ]));
        } else {
          return r;
        }
      })));
    },
    executeQuery(e, r) {
      var t = p.createRequestOperation("query", e, r);
      return p.executeRequestOperation(t);
    },
    executeSubscription(e, r) {
      var t = p.createRequestOperation("subscription", e, r);
      return p.executeRequestOperation(t);
    },
    executeMutation(e, r) {
      var t = p.createRequestOperation("mutation", e, r);
      return p.executeRequestOperation(t);
    },
    readQuery(e, r, t) {
      var n = null;
      y((e => {
        n = e;
      }))(p.query(e, r, t)).unsubscribe();
      return n;
    },
    query: (e, r, t) => p.executeQuery(v(e, r), t),
    subscription: (e, r, t) => p.executeSubscription(v(e, r), t),
    mutation: (e, r, t) => p.executeMutation(v(e, r), t)
  });
  var d = noop;
  if ("production" !== process.env.NODE_ENV) {
    var {next: f, source: x} = D();
    p.subscribeToDebugTarget = e => y(e)(x);
    d = f;
  }
  var w = composeExchanges(e.exchanges);
  var E = N(w({
    client: p,
    dispatchDebug: d,
    forward: fallbackExchange({
      dispatchDebug: d
    })
  })(s.source));
  V(E);
  return p;
};

var Q = C;

export { C as Client, a as CombinedError, cacheExchange, composeExchanges, Q as createClient, v as createRequest, debugExchange, mapExchange as errorExchange, fetchExchange, formatDocument, gql, s as makeErrorResult, makeOperation, n as makeResult, mapExchange, c as mergeResultPatch, ssrExchange, t as stringifyDocument, subscriptionExchange };
//# sourceMappingURL=urql-core.mjs.map
