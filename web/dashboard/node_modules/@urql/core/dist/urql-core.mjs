import { visit as e } from "graphql/language/visitor.mjs";

import { Kind as r } from "graphql/language/kinds.mjs";

import { print as n } from "graphql/language/printer.mjs";

import { k as t, _ as o, s as u, C as c, m as s, a as p, b as l, c as d, d as v, e as y, g as k, f as g } from "./01e1547d.mjs";

export { C as CombinedError, f as createRequest, j as getOperationName, a as makeErrorResult, m as makeResult, i as mergeResultPatch, h as stringifyVariables } from "./01e1547d.mjs";

import { toPromise as x, take as E, filter as O, share as b, map as w, tap as q, merge as N, mergeMap as D, takeUntil as _, make as S, onPush as R, makeSubject as T, subscribe as P, onEnd as M, onStart as A, publish as V, switchMap as I, fromValue as F } from "wonka";

function collectTypes(e, r) {
  if (Array.isArray(e)) {
    for (var n = 0; n < e.length; n++) {
      collectTypes(e[n], r);
    }
  } else if ("object" == typeof e && null !== e) {
    for (var t in e) {
      if ("__typename" === t && "string" == typeof e[t]) {
        r[e[t]] = 0;
      } else {
        collectTypes(e[t], r);
      }
    }
  }
  return r;
}

function collectTypesFromResponse(e) {
  return Object.keys(collectTypes(e, {}));
}

var formatNode = function(e) {
  if (e.selectionSet && !e.selectionSet.selections.some((function(e) {
    return e.kind === r.FIELD && "__typename" === e.name.value && !e.alias;
  }))) {
    return o({}, e, {
      selectionSet: o({}, e.selectionSet, {
        selections: e.selectionSet.selections.concat([ {
          kind: r.FIELD,
          name: {
            kind: r.NAME,
            value: "__typename"
          }
        } ])
      })
    });
  }
};

var Q = new Map;

function formatDocument(r) {
  var n = t(r);
  var a = Q.get(n.__key);
  if (!a) {
    a = e(n, {
      Field: formatNode,
      InlineFragment: formatNode
    });
    Object.defineProperty(a, "__key", {
      value: n.__key,
      enumerable: !1
    });
    Q.set(n.__key, a);
  }
  return a;
}

function maskTypename(e) {
  if (!e || "object" != typeof e) {
    return e;
  }
  return Object.keys(e).reduce((function(r, n) {
    var t = e[n];
    if ("__typename" === n) {
      Object.defineProperty(r, "__typename", {
        enumerable: !1,
        value: t
      });
    } else if (Array.isArray(t)) {
      r[n] = t.map(maskTypename);
    } else if (t && "object" == typeof t && "__typename" in t) {
      r[n] = maskTypename(t);
    } else {
      r[n] = t;
    }
    return r;
  }), Array.isArray(e) ? [] : {});
}

function withPromise(e) {
  e.toPromise = function() {
    return x(E(1)(O((function(e) {
      return !e.stale && !e.hasNext;
    }))(e)));
  };
  return e;
}

function makeOperation(e, r, n) {
  if (!n) {
    n = r.context;
  }
  return {
    key: r.key,
    query: r.query,
    variables: r.variables,
    kind: e,
    context: n
  };
}

function addMetadata(e, r) {
  return makeOperation(e.kind, e, o({}, e.context, {
    meta: o({}, e.context.meta, r)
  }));
}

function noop() {}

function applyDefinitions(e, n, t) {
  for (var a = 0; a < t.length; a++) {
    if (t[a].kind === r.FRAGMENT_DEFINITION) {
      var o = t[a].name.value;
      var i = u(t[a]);
      if (!e.has(o)) {
        e.set(o, i);
        n.push(t[a]);
      } else if ("production" !== process.env.NODE_ENV && e.get(o) !== i) {
        console.warn("[WARNING: Duplicate Fragment] A fragment with name `" + o + "` already exists in this document.\nWhile fragment names may not be unique across your source, each name must be unique per document.");
      }
    } else {
      n.push(t[a]);
    }
  }
}

function gql() {
  var e = arguments;
  var n = new Map;
  var a = [];
  var o = [];
  var i = Array.isArray(arguments[0]) ? arguments[0][0] : arguments[0] || "";
  for (var u = 1; u < arguments.length; u++) {
    var c = e[u];
    if (c && c.definitions) {
      o.push.apply(o, c.definitions);
    } else {
      i += c;
    }
    i += e[0][u];
  }
  applyDefinitions(n, a, t(i).definitions);
  applyDefinitions(n, a, o);
  return t({
    kind: r.DOCUMENT,
    definitions: a
  });
}

function shouldSkip(e) {
  var r = e.kind;
  return "mutation" !== r && "query" !== r;
}

function cacheExchange(e) {
  var r = e.forward;
  var n = e.client;
  var t = e.dispatchDebug;
  var a = new Map;
  var i = Object.create(null);
  function mapTypeNames(e) {
    var r = makeOperation(e.kind, e);
    r.query = formatDocument(e.query);
    return r;
  }
  function isOperationCached(e) {
    var r = e.context.requestPolicy;
    return "query" === e.kind && "network-only" !== r && ("cache-only" === r || a.has(e.key));
  }
  return function(e) {
    var u = b(e);
    var c = w((function(e) {
      var r = a.get(e.key);
      "production" !== process.env.NODE_ENV && t(o({}, {
        operation: e
      }, r ? {
        type: "cacheHit",
        message: "The result was successfully retried from the cache"
      } : {
        type: "cacheMiss",
        message: "The result could not be retrieved from the cache"
      }));
      var i = o({}, r, {
        operation: addMetadata(e, {
          cacheOutcome: r ? "hit" : "miss"
        })
      });
      if ("cache-and-network" === e.context.requestPolicy) {
        i.stale = !0;
        reexecuteOperation(n, e);
      }
      return i;
    }))(O((function(e) {
      return !shouldSkip(e) && isOperationCached(e);
    }))(u));
    var s = q((function(e) {
      var r = e.operation;
      if (!r) {
        return;
      }
      var o = collectTypesFromResponse(e.data).concat(r.context.additionalTypenames || []);
      if ("mutation" === e.operation.kind) {
        var u = new Set;
        "production" !== process.env.NODE_ENV && t({
          type: "cacheInvalidation",
          message: "The following typenames have been invalidated: " + o,
          operation: r,
          data: {
            typenames: o,
            response: e
          },
          source: "cacheExchange"
        });
        for (var c = 0; c < o.length; c++) {
          var s = o[c];
          var f = i[s] || (i[s] = new Set);
          f.forEach((function(e) {
            u.add(e);
          }));
          f.clear();
        }
        u.forEach((function(e) {
          if (a.has(e)) {
            r = a.get(e).operation;
            a.delete(e);
            reexecuteOperation(n, r);
          }
        }));
      } else if ("query" === r.kind && e.data) {
        a.set(r.key, e);
        for (var p = 0; p < o.length; p++) {
          var l = o[p];
          (i[l] || (i[l] = new Set)).add(r.key);
        }
      }
    }))(r(O((function(e) {
      return "query" !== e.kind || "cache-only" !== e.context.requestPolicy;
    }))(w((function(e) {
      return addMetadata(e, {
        cacheOutcome: "miss"
      });
    }))(N([ w(mapTypeNames)(O((function(e) {
      return !shouldSkip(e) && !isOperationCached(e);
    }))(u)), O((function(e) {
      return shouldSkip(e);
    }))(u) ])))));
    return N([ c, s ]);
  };
}

function reexecuteOperation(e, r) {
  return e.reexecuteOperation(makeOperation(r.kind, r, o({}, r.context, {
    requestPolicy: "network-only"
  })));
}

var G = new Set;

function ssrExchange(e) {
  var r = !(!e || !e.staleWhileRevalidate);
  var n = !(!e || !e.includeExtensions);
  var t = {};
  var a = [];
  function invalidate(e) {
    a.push(e.operation.key);
    if (1 === a.length) {
      Promise.resolve().then((function() {
        var e;
        while (e = a.shift()) {
          t[e] = null;
        }
      }));
    }
  }
  var ssr = function(a) {
    var o = a.client;
    var i = a.forward;
    return function(a) {
      var u = e && "boolean" == typeof e.isClient ? !!e.isClient : !o.suspense;
      var s = b(a);
      var f = i(O((function(e) {
        return !t[e.key] || !!t[e.key].hasNext;
      }))(s));
      var p = w((function(e) {
        var a = function deserializeResult(e, r, n) {
          return {
            operation: e,
            data: r.data ? JSON.parse(r.data) : void 0,
            extensions: n && r.extensions ? JSON.parse(r.extensions) : void 0,
            error: r.error ? new c({
              networkError: r.error.networkError ? new Error(r.error.networkError) : void 0,
              graphQLErrors: r.error.graphQLErrors
            }) : void 0,
            hasNext: r.hasNext
          };
        }(e, t[e.key], n);
        if (r && !G.has(e.key)) {
          a.stale = !0;
          G.add(e.key);
          reexecuteOperation(o, e);
        }
        return a;
      }))(O((function(e) {
        return !!t[e.key];
      }))(s));
      if (!u) {
        f = q((function(e) {
          var r = e.operation;
          if ("mutation" !== r.kind) {
            var a = function serializeResult(e, r) {
              var n = e.hasNext;
              var t = e.data;
              var a = e.extensions;
              var o = e.error;
              var i = {};
              if (void 0 !== t) {
                i.data = JSON.stringify(t);
              }
              if (r && void 0 !== a) {
                i.extensions = JSON.stringify(a);
              }
              if (n) {
                i.hasNext = !0;
              }
              if (o) {
                i.error = {
                  graphQLErrors: o.graphQLErrors.map((function(e) {
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
                if (o.networkError) {
                  i.error.networkError = "" + o.networkError;
                }
              }
              return i;
            }(e, n);
            t[r.key] = a;
          }
        }))(f);
      } else {
        p = q(invalidate)(p);
      }
      return N([ f, p ]);
    };
  };
  ssr.restoreData = function(e) {
    for (var r in e) {
      if (null !== t[r]) {
        t[r] = e[r];
      }
    }
  };
  ssr.extractData = function() {
    var e = {};
    for (var r in t) {
      if (null != t[r]) {
        e[r] = t[r];
      }
    }
    return e;
  };
  if (e && e.initialState) {
    ssr.restoreData(e.initialState);
  }
  return ssr;
}

function subscriptionExchange(e) {
  var r = e.forwardSubscription;
  var t = e.enableAllOperations;
  return function(e) {
    var a = e.client;
    var i = e.forward;
    function isSubscriptionOperation(e) {
      var r = e.kind;
      return "subscription" === r || !!t && ("query" === r || "mutation" === r);
    }
    return function(e) {
      var t = b(e);
      var u = D((function(e) {
        var i = e.key;
        var u = O((function(e) {
          return "teardown" === e.kind && e.key === i;
        }))(t);
        return _(u)(function createSubscriptionSource(e) {
          var t = r({
            key: e.key.toString(36),
            query: n(e.query),
            variables: e.variables,
            context: o({}, e.context)
          });
          return S((function(r) {
            var n = r.next;
            var o = r.complete;
            var i = !1;
            var u;
            Promise.resolve().then((function() {
              if (i) {
                return;
              }
              u = t.subscribe({
                next: function(r) {
                  return n(s(e, r));
                },
                error: function(r) {
                  return n(p(e, r));
                },
                complete: function() {
                  if (!i) {
                    i = !0;
                    if ("subscription" === e.kind) {
                      a.reexecuteOperation(makeOperation("teardown", e, e.context));
                    }
                    o();
                  }
                }
              });
            }));
            return function() {
              i = !0;
              if (u) {
                u.unsubscribe();
              }
            };
          }));
        }(e));
      }))(O(isSubscriptionOperation)(t));
      var c = i(O((function(e) {
        return !isSubscriptionOperation(e);
      }))(t));
      return N([ u, c ]);
    };
  };
}

function debugExchange(e) {
  var r = e.forward;
  if ("production" === process.env.NODE_ENV) {
    return function(e) {
      return r(e);
    };
  } else {
    return function(e) {
      return q((function(e) {
        return console.log("[Exchange debug]: Completed operation: ", e);
      }))(r(q((function(e) {
        return console.log("[Exchange debug]: Incoming operation: ", e);
      }))(e)));
    };
  }
}

function dedupExchange(e) {
  var r = e.forward;
  var n = e.dispatchDebug;
  var t = new Set;
  function filterIncomingOperation(e) {
    var r = e.key;
    var a = e.kind;
    if ("teardown" === a) {
      t.delete(r);
      return !0;
    }
    if ("query" !== a && "subscription" !== a) {
      return !0;
    }
    var o = t.has(r);
    t.add(r);
    if (o) {
      "production" !== process.env.NODE_ENV && n({
        type: "dedup",
        message: "An operation has been deduped.",
        operation: e,
        source: "dedupExchange"
      });
    }
    return !o;
  }
  function afterOperationResult(e) {
    if (!e.hasNext) {
      t.delete(e.operation.key);
    }
  }
  return function(e) {
    var n = O(filterIncomingOperation)(e);
    return q(afterOperationResult)(r(n));
  };
}

function fetchExchange(e) {
  var r = e.forward;
  var n = e.dispatchDebug;
  return function(e) {
    var t = b(e);
    var a = D((function(e) {
      var r = e.key;
      var a = O((function(e) {
        return "teardown" === e.kind && e.key === r;
      }))(t);
      var o = l(e);
      var i = d(e, o);
      var u = v(e, o);
      "production" !== process.env.NODE_ENV && n({
        type: "fetchRequest",
        message: "A fetch request is being executed.",
        operation: e,
        data: {
          url: i,
          fetchOptions: u
        },
        source: "fetchExchange"
      });
      return R((function(r) {
        var t = !r.data ? r.error : void 0;
        "production" !== process.env.NODE_ENV && n({
          type: t ? "fetchError" : "fetchSuccess",
          message: "A " + (t ? "failed" : "successful") + " fetch response has been returned.",
          operation: e,
          data: {
            url: i,
            fetchOptions: u,
            value: t || r
          },
          source: "fetchExchange"
        });
      }))(_(a)(y(e, i, u)));
    }))(O((function(e) {
      return "query" === e.kind || "mutation" === e.kind;
    }))(t));
    var o = r(O((function(e) {
      return "query" !== e.kind && "mutation" !== e.kind;
    }))(t));
    return N([ a, o ]);
  };
}

function fallbackExchange(e) {
  var r = e.dispatchDebug;
  return function(e) {
    return O((function() {
      return !1;
    }))(q((function(e) {
      if ("teardown" !== e.kind && "production" !== process.env.NODE_ENV) {
        var n = 'No exchange has handled operations of kind "' + e.kind + "\". Check whether you've added an exchange responsible for these operations.";
        "production" !== process.env.NODE_ENV && r({
          type: "fallbackCatch",
          message: n,
          operation: e,
          source: "fallbackExchange"
        });
        console.warn(n);
      }
    }))(e));
  };
}

var L = fallbackExchange({
  dispatchDebug: noop
});

function composeExchanges(e) {
  return function(r) {
    var n = r.client;
    var t = r.dispatchDebug;
    return e.reduceRight((function(e, r) {
      return r({
        client: n,
        forward: e,
        dispatchDebug: function dispatchDebug$1(e) {
          "production" !== process.env.NODE_ENV && t(o({}, {
            timestamp: Date.now(),
            source: r.name
          }, e));
        }
      });
    }), r.forward);
  };
}

function errorExchange(e) {
  var r = e.onError;
  return function(e) {
    var n = e.forward;
    return function(e) {
      return q((function(e) {
        var n = e.error;
        if (n) {
          r(n, e.operation);
        }
      }))(n(e));
    };
  };
}

var J = [ dedupExchange, cacheExchange, fetchExchange ];

var W = function Client(e) {
  if ("production" !== process.env.NODE_ENV && !e.url) {
    throw new Error("You are creating an urql-client without a url.");
  }
  var r = new Map;
  var n = new Map;
  var t = [];
  var a = T();
  var i = a.source;
  var u = a.next;
  var c = !1;
  function dispatchOperation(e) {
    c = !0;
    if (e) {
      u(e);
    }
    while (e = t.shift()) {
      u(e);
    }
    c = !1;
  }
  function makeResultSource(e) {
    var a = O((function(r) {
      return r.operation.kind === e.kind && r.operation.key === e.key;
    }))(y);
    if (f.maskTypename) {
      a = w((function(e) {
        return o({}, e, {
          data: maskTypename(e.data)
        });
      }))(a);
    }
    if ("mutation" === e.kind) {
      return E(1)(A((function() {
        return dispatchOperation(e);
      }))(a));
    }
    return b(M((function() {
      r.delete(e.key);
      n.delete(e.key);
      for (var a = t.length - 1; a >= 0; a--) {
        if (t[a].key === e.key) {
          t.splice(a, 1);
        }
      }
      dispatchOperation(makeOperation("teardown", e, e.context));
    }))(R((function(n) {
      r.set(e.key, n);
    }))(I((function(r) {
      if ("query" !== e.kind || r.stale) {
        return F(r);
      }
      return N([ F(r), w((function() {
        return o({}, r, {
          stale: !0
        });
      }))(E(1)(O((function(r) {
        return "query" === r.kind && r.key === e.key && "cache-only" !== r.context.requestPolicy;
      }))(i))) ]);
    }))(_(O((function(r) {
      return "teardown" === r.kind && r.key === e.key;
    }))(i))(a)))));
  }
  var s = this instanceof Client ? this : Object.create(Client.prototype);
  var f = o(s, {
    url: e.url,
    fetchOptions: e.fetchOptions,
    fetch: e.fetch,
    suspense: !!e.suspense,
    requestPolicy: e.requestPolicy || "cache-first",
    preferGetMethod: !!e.preferGetMethod,
    maskTypename: !!e.maskTypename,
    operations$: i,
    reexecuteOperation: function reexecuteOperation(e) {
      if ("mutation" === e.kind || n.has(e.key)) {
        t.push(e);
        if (!c) {
          Promise.resolve().then(dispatchOperation);
        }
      }
    },
    createOperationContext: function createOperationContext(e) {
      if (!e) {
        e = {};
      }
      return o({}, {
        url: f.url,
        fetchOptions: f.fetchOptions,
        fetch: f.fetch,
        preferGetMethod: f.preferGetMethod
      }, e, {
        suspense: e.suspense || !1 !== e.suspense && f.suspense,
        requestPolicy: e.requestPolicy || f.requestPolicy
      });
    },
    createRequestOperation: function createRequestOperation(e, r, n) {
      var t = k(r.query);
      if ("production" !== process.env.NODE_ENV && "teardown" !== e && t !== e) {
        throw new Error('Expected operation of type "' + e + '" but found "' + t + '"');
      }
      return makeOperation(e, r, f.createOperationContext(n));
    },
    executeRequestOperation: function executeRequestOperation(e) {
      if ("mutation" === e.kind) {
        return makeResultSource(e);
      }
      return S((function(t) {
        var a = n.get(e.key);
        if (!a) {
          n.set(e.key, a = makeResultSource(e));
        }
        var i = "cache-and-network" === e.context.requestPolicy || "network-only" === e.context.requestPolicy;
        return P(t.next)(M(t.complete)(A((function() {
          var n = r.get(e.key);
          if ("subscription" === e.kind) {
            return dispatchOperation(e);
          } else if (i) {
            dispatchOperation(e);
          }
          if (null != n && n === r.get(e.key)) {
            t.next(i ? o({}, n, {
              stale: !0
            }) : n);
          } else if (!i) {
            dispatchOperation(e);
          }
        }))(a))).unsubscribe;
      }));
    },
    executeQuery: function executeQuery(e, r) {
      var n = f.createRequestOperation("query", e, r);
      return f.executeRequestOperation(n);
    },
    executeSubscription: function executeSubscription(e, r) {
      var n = f.createRequestOperation("subscription", e, r);
      return f.executeRequestOperation(n);
    },
    executeMutation: function executeMutation(e, r) {
      var n = f.createRequestOperation("mutation", e, r);
      return f.executeRequestOperation(n);
    },
    query: function query(e, r, n) {
      if (!n || "boolean" != typeof n.suspense) {
        n = o({}, n, {
          suspense: !1
        });
      }
      return withPromise(f.executeQuery(g(e, r), n));
    },
    readQuery: function readQuery(e, r, n) {
      var t = null;
      P((function(e) {
        t = e;
      }))(f.query(e, r, n)).unsubscribe();
      return t;
    },
    subscription: function subscription(e, r, n) {
      return f.executeSubscription(g(e, r), n);
    },
    mutation: function mutation(e, r, n) {
      return withPromise(f.executeMutation(g(e, r), n));
    }
  });
  var p = noop;
  if ("production" !== process.env.NODE_ENV) {
    var l = T();
    var d = l.next;
    var h = l.source;
    f.subscribeToDebugTarget = function(e) {
      return P(e)(h);
    };
    p = d;
  }
  var v = composeExchanges(void 0 !== e.exchanges ? e.exchanges : J);
  var y = b(v({
    client: f,
    dispatchDebug: p,
    forward: fallbackExchange({
      dispatchDebug: p
    })
  })(i));
  V(y);
  return f;
};

var z = W;

export { W as Client, cacheExchange, composeExchanges, z as createClient, debugExchange, dedupExchange, J as defaultExchanges, errorExchange, L as fallbackExchangeIO, fetchExchange, formatDocument, gql, makeOperation, maskTypename, ssrExchange, subscriptionExchange };
//# sourceMappingURL=urql-core.mjs.map
