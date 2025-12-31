import { Kind as e } from "@0no-co/graphql.web";

import { k as r, s as t, m as n, C as a, a as o, b as s, c, d as u, e as p, f as d, g as v, h as f } from "./urql-core-chunk.mjs";

export { i as stringifyVariables } from "./urql-core-chunk.mjs";

import { toPromise as l, take as h, filter as y, subscribe as k, map as m, tap as x, merge as w, mergeMap as g, takeUntil as E, make as O, onPush as N, share as b, fromPromise as _, fromValue as q, makeSubject as D, lazy as S, onStart as P, switchMap as R, publish as V, takeWhile as A, onEnd as M } from "wonka";

var collectTypes = (e, r) => {
  if (Array.isArray(e)) {
    for (var t of e) {
      collectTypes(t, r);
    }
  } else if ("object" == typeof e && null !== e) {
    for (var n in e) {
      if ("__typename" === n && "string" == typeof e[n]) {
        r.add(e[n]);
      } else {
        collectTypes(e[n], r);
      }
    }
  }
  return r;
};

var formatNode = r => {
  if ("definitions" in r) {
    var t = [];
    for (var n of r.definitions) {
      var a = formatNode(n);
      t.push(a);
    }
    return {
      ...r,
      definitions: t
    };
  }
  if ("directives" in r && r.directives && r.directives.length) {
    var o = [];
    var i = {};
    for (var s of r.directives) {
      var c = s.name.value;
      if ("_" !== c[0]) {
        o.push(s);
      } else {
        c = c.slice(1);
      }
      i[c] = s;
    }
    r = {
      ...r,
      directives: o,
      _directives: i
    };
  }
  if ("selectionSet" in r) {
    var u = [];
    var p = r.kind === e.OPERATION_DEFINITION;
    if (r.selectionSet) {
      for (var d of r.selectionSet.selections || []) {
        p = p || d.kind === e.FIELD && "__typename" === d.name.value && !d.alias;
        var v = formatNode(d);
        u.push(v);
      }
      if (!p) {
        u.push({
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
          selections: u
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

var maskTypename = (e, r) => {
  if (!e || "object" != typeof e) {
    return e;
  } else if (Array.isArray(e)) {
    return e.map((e => maskTypename(e)));
  } else if (e && "object" == typeof e && (r || "__typename" in e)) {
    var t = {};
    for (var n in e) {
      if ("__typename" === n) {
        Object.defineProperty(t, "__typename", {
          enumerable: !1,
          value: e.__typename
        });
      } else {
        t[n] = maskTypename(e[n]);
      }
    }
    return t;
  } else {
    return e;
  }
};

function withPromise(e) {
  var source$ = r => e(r);
  source$.toPromise = () => l(h(1)(y((e => !e.stale && !e.hasNext))(source$)));
  source$.then = (e, r) => source$.toPromise().then(e, r);
  source$.subscribe = e => k(e)(source$);
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
  var o = [];
  var i = [];
  var s = Array.isArray(n) ? n[0] : n || "";
  for (var c = 1; c < arguments.length; c++) {
    var u = arguments[c];
    if (u && u.definitions) {
      i.push(u);
    } else {
      s += u;
    }
    s += arguments[0][c];
  }
  i.unshift(r(s));
  for (var p of i) {
    for (var d of p.definitions) {
      if (d.kind === e.FRAGMENT_DEFINITION) {
        var v = d.name.value;
        var f = t(d);
        if (!a.has(v)) {
          a.set(v, f);
          o.push(d);
        } else if ("production" !== process.env.NODE_ENV && a.get(v) !== f) {
          console.warn("[WARNING: Duplicate Fragment] A fragment with name `" + v + "` already exists in this document.\nWhile fragment names may not be unique across your source, each name must be unique per document.");
        }
      } else {
        o.push(d);
      }
    }
  }
  return r({
    kind: e.DOCUMENT,
    definitions: o
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
  var o = new Map;
  var isOperationCached = e => "query" === e.kind && "network-only" !== e.context.requestPolicy && ("cache-only" === e.context.requestPolicy || a.has(e.key));
  return i => {
    var s = m((e => {
      var o = a.get(e.key);
      "production" !== process.env.NODE_ENV && t({
        operation: e,
        ...o ? {
          type: "cacheHit",
          message: "The result was successfully retried from the cache"
        } : {
          type: "cacheMiss",
          message: "The result could not be retrieved from the cache"
        },
        source: "cacheExchange"
      });
      var i = o || n(e, {
        data: null
      });
      i = {
        ...i,
        operation: "production" !== process.env.NODE_ENV ? addMetadata(e, {
          cacheOutcome: o ? "hit" : "miss"
        }) : e
      };
      if ("cache-and-network" === e.context.requestPolicy) {
        i.stale = !0;
        reexecuteOperation(r, e);
      }
      return i;
    }))(y((e => !shouldSkip(e) && isOperationCached(e)))(i));
    var c = x((e => {
      var {operation: n} = e;
      if (!n) {
        return;
      }
      var i = n.context.additionalTypenames || [];
      if ("subscription" !== e.operation.kind) {
        i = (e => [ ...collectTypes(e, new Set) ])(e.data).concat(i);
      }
      if ("mutation" === e.operation.kind || "subscription" === e.operation.kind) {
        var s = new Set;
        "production" !== process.env.NODE_ENV && t({
          type: "cacheInvalidation",
          message: `The following typenames have been invalidated: ${i}`,
          operation: n,
          data: {
            typenames: i,
            response: e
          },
          source: "cacheExchange"
        });
        for (var c = 0; c < i.length; c++) {
          var u = i[c];
          var p = o.get(u);
          if (!p) {
            o.set(u, p = new Set);
          }
          for (var d of p.values()) {
            s.add(d);
          }
          p.clear();
        }
        for (var v of s.values()) {
          if (a.has(v)) {
            n = a.get(v).operation;
            a.delete(v);
            reexecuteOperation(r, n);
          }
        }
      } else if ("query" === n.kind && e.data) {
        a.set(n.key, e);
        for (var f = 0; f < i.length; f++) {
          var l = i[f];
          var h = o.get(l);
          if (!h) {
            o.set(l, h = new Set);
          }
          h.add(n.key);
        }
      }
    }))(e(y((e => "query" !== e.kind || "cache-only" !== e.context.requestPolicy))(m((e => "production" !== process.env.NODE_ENV ? addMetadata(e, {
      cacheOutcome: "miss"
    }) : e))(w([ m(mapTypeNames)(y((e => !shouldSkip(e) && !isOperationCached(e)))(i)), y((e => shouldSkip(e)))(i) ])))));
    return w([ s, c ]);
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
  var o = [];
  var invalidate = e => {
    o.push(e.operation.key);
    if (1 === o.length) {
      Promise.resolve().then((() => {
        var e;
        while (e = o.shift()) {
          n[e] = null;
        }
      }));
    }
  };
  var ssr = ({client: o, forward: i}) => s => {
    var c = e && "boolean" == typeof e.isClient ? !!e.isClient : !o.suspense;
    var u = i(m(mapTypeNames)(y((e => "teardown" === e.kind || !n[e.key] || !!n[e.key].hasNext || "network-only" === e.context.requestPolicy))(s)));
    var p = m((e => {
      var i = ((e, r, t) => ({
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
        i.stale = !0;
        T.add(e.key);
        reexecuteOperation(o, e);
      }
      return {
        ...i,
        operation: "production" !== process.env.NODE_ENV ? addMetadata(e, {
          cacheOutcome: "hit"
        }) : e
      };
    }))(y((e => "teardown" !== e.kind && !!n[e.key] && "network-only" !== e.context.requestPolicy))(s));
    if (!c) {
      u = x((e => {
        var {operation: r} = e;
        if ("mutation" !== r.kind) {
          var a = ((e, r) => {
            var t = {
              data: JSON.stringify(e.data),
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
    return w([ u, p ]);
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
    var t = g((t => {
      var {key: i} = t;
      var u = y((e => "teardown" === e.kind && e.key === i))(r);
      return E(u)((r => {
        var t = e(o(r), r);
        return O((e => {
          var o = !1;
          var i;
          var u;
          function nextResult(t) {
            e.next(u = u ? c(u, t) : n(r, t));
          }
          Promise.resolve().then((() => {
            if (o) {
              return;
            }
            i = t.subscribe({
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
                if (!o) {
                  o = !0;
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
            o = !0;
            if (i) {
              i.unsubscribe();
            }
          };
        }));
      })(t));
    }))(y((e => "teardown" !== e.kind && u(e)))(r));
    var p = i(y((e => "teardown" === e.kind || !u(e)))(r));
    return w([ t, p ]);
  };
};

var debugExchange = ({forward: e}) => {
  if ("production" === process.env.NODE_ENV) {
    return r => e(r);
  } else {
    return r => x((e => console.log("[Exchange debug]: Completed operation: ", e)))(e(x((e => console.log("[Exchange debug]: Incoming operation: ", e)))(r)));
  }
};

var dedupExchange = ({forward: e}) => r => e(r);

var fetchExchange = ({forward: e, dispatchDebug: r}) => t => {
  var n = g((e => {
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
    var s = E(y((r => "teardown" === r.kind && r.key === e.key))(t))(d(e, a, i));
    if ("production" !== process.env.NODE_ENV) {
      return N((t => {
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
  }))(y((e => "teardown" !== e.kind && ("subscription" !== e.kind || !!e.context.fetchSubscriptions)))(t));
  var a = e(y((e => "teardown" === e.kind || "subscription" === e.kind && !e.context.fetchSubscriptions))(t));
  return w([ n, a ]);
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
      return b(e(b(r)));
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

var mapExchange = ({onOperation: e, onResult: r, onError: t}) => ({forward: n}) => a => g((e => {
  if (t && e.error) {
    t(e.error, e.operation);
  }
  var n = r && r(e) || e;
  return "then" in n ? _(n) : q(n);
}))(n(g((r => {
  var t = e && e(r) || r;
  return "then" in t ? _(t) : q(t);
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
  return y((e => !1))(r);
};

var C = function Client(e) {
  if ("production" !== process.env.NODE_ENV && !e.url) {
    throw new Error("You are creating an urql-client without a url.");
  }
  var r = 0;
  var t = new Map;
  var n = new Map;
  var a = new Set;
  var o = [];
  var i = {
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
      while (c && (e = o.shift())) {
        nextOperation(e);
      }
      c = !1;
    }
  }
  var makeResultSource = r => {
    var i = E(y((e => "teardown" === e.kind && e.key === r.key))(s.source))(y((e => e.operation.kind === r.kind && e.operation.key === r.key && (!e.operation.context._instance || e.operation.context._instance === r.context._instance)))(O));
    if (e.maskTypename) {
      i = m((e => ({
        ...e,
        data: maskTypename(e.data, !0)
      })))(i);
    }
    if ("query" !== r.kind) {
      i = A((e => !!e.hasNext), !0)(i);
    } else {
      i = R((e => {
        var t = q(e);
        return e.stale || e.hasNext ? t : w([ t, m((() => {
          e.stale = !0;
          return e;
        }))(h(1)(y((e => e.key === r.key))(s.source))) ]);
      }))(i);
    }
    if ("mutation" !== r.kind) {
      i = M((() => {
        a.delete(r.key);
        t.delete(r.key);
        n.delete(r.key);
        c = !1;
        for (var e = o.length - 1; e >= 0; e--) {
          if (o[e].key === r.key) {
            o.splice(e, 1);
          }
        }
        nextOperation(makeOperation("teardown", r, r.context));
      }))(N((e => {
        if (e.stale) {
          for (var n of o) {
            if (n.key === e.operation.key) {
              a.delete(n.key);
              break;
            }
          }
        } else if (!e.hasNext) {
          a.delete(r.key);
        }
        t.set(r.key, e);
      }))(i));
    } else {
      i = P((() => {
        nextOperation(r);
      }))(i);
    }
    return b(i);
  };
  var u = this instanceof Client ? this : Object.create(Client.prototype);
  var p = Object.assign(u, {
    suspense: !!e.suspense,
    operations$: s.source,
    reexecuteOperation(e) {
      if ("teardown" === e.kind) {
        dispatchOperation(e);
      } else if ("mutation" === e.kind || n.has(e.key)) {
        var r = !1;
        for (var t = 0; t < o.length; t++) {
          r = r || o[t].key === e.key;
        }
        if (!r) {
          a.delete(e.key);
        }
        o.push(e);
        Promise.resolve().then(dispatchOperation);
      }
    },
    createRequestOperation(e, t, n) {
      if (!n) {
        n = {};
      }
      var a;
      if ("production" !== process.env.NODE_ENV && "teardown" !== e && (a = v(t.query)) !== e) {
        throw new Error(`Expected operation of type "${e}" but found "${a}"`);
      }
      return makeOperation(e, t, {
        _instance: "mutation" === e ? r = r + 1 | 0 : void 0,
        ...i,
        ...n,
        requestPolicy: n.requestPolicy || i.requestPolicy,
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
          return R(q)(w([ r, y((r => r === t.get(e.key)))(q(a)) ]));
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
      k((e => {
        n = e;
      }))(p.query(e, r, t)).unsubscribe();
      return n;
    },
    query: (e, r, t) => p.executeQuery(f(e, r), t),
    subscription: (e, r, t) => p.executeSubscription(f(e, r), t),
    mutation: (e, r, t) => p.executeMutation(f(e, r), t)
  });
  var d = noop;
  if ("production" !== process.env.NODE_ENV) {
    var {next: l, source: x} = D();
    p.subscribeToDebugTarget = e => k(e)(x);
    d = l;
  }
  var g = composeExchanges(e.exchanges);
  var O = b(g({
    client: p,
    dispatchDebug: d,
    forward: fallbackExchange({
      dispatchDebug: d
    })
  })(s.source));
  V(O);
  return p;
};

var j = C;

export { C as Client, a as CombinedError, cacheExchange, composeExchanges, j as createClient, f as createRequest, debugExchange, dedupExchange, mapExchange as errorExchange, fetchExchange, formatDocument, gql, s as makeErrorResult, makeOperation, n as makeResult, mapExchange, maskTypename, c as mergeResultPatch, ssrExchange, t as stringifyDocument, subscriptionExchange };
//# sourceMappingURL=urql-core.mjs.map
