var e = require("graphql");

var t = require("./878a8849.js");

var r = require("wonka");

function collectTypes(e, t) {
  if (Array.isArray(e)) {
    for (var r = 0; r < e.length; r++) {
      collectTypes(e[r], t);
    }
  } else if ("object" == typeof e && null !== e) {
    for (var n in e) {
      if ("__typename" === n && "string" == typeof e[n]) {
        t[e[n]] = 0;
      } else {
        collectTypes(e[n], t);
      }
    }
  }
  return t;
}

function collectTypesFromResponse(e) {
  return Object.keys(collectTypes(e, {}));
}

var formatNode = function(r) {
  if (r.selectionSet && !r.selectionSet.selections.some((function(t) {
    return t.kind === e.Kind.FIELD && "__typename" === t.name.value && !t.alias;
  }))) {
    return t._extends({}, r, {
      selectionSet: t._extends({}, r.selectionSet, {
        selections: r.selectionSet.selections.concat([ {
          kind: e.Kind.FIELD,
          name: {
            kind: e.Kind.NAME,
            value: "__typename"
          }
        } ])
      })
    });
  }
};

var n = new Map;

function formatDocument(r) {
  var a = t.keyDocument(r);
  var o = n.get(a.__key);
  if (!o) {
    o = e.visit(a, {
      Field: formatNode,
      InlineFragment: formatNode
    });
    Object.defineProperty(o, "__key", {
      value: a.__key,
      enumerable: !1
    });
    n.set(a.__key, o);
  }
  return o;
}

function maskTypename(e) {
  if (!e || "object" != typeof e) {
    return e;
  }
  return Object.keys(e).reduce((function(t, r) {
    var n = e[r];
    if ("__typename" === r) {
      Object.defineProperty(t, "__typename", {
        enumerable: !1,
        value: n
      });
    } else if (Array.isArray(n)) {
      t[r] = n.map(maskTypename);
    } else if (n && "object" == typeof n && "__typename" in n) {
      t[r] = maskTypename(n);
    } else {
      t[r] = n;
    }
    return t;
  }), Array.isArray(e) ? [] : {});
}

function withPromise(e) {
  e.toPromise = function() {
    return r.toPromise(r.take(1)(r.filter((function(e) {
      return !e.stale && !e.hasNext;
    }))(e)));
  };
  return e;
}

function makeOperation(e, t, r) {
  if (!r) {
    r = t.context;
  }
  return {
    key: t.key,
    query: t.query,
    variables: t.variables,
    kind: e,
    context: r
  };
}

function addMetadata(e, r) {
  return makeOperation(e.kind, e, t._extends({}, e.context, {
    meta: t._extends({}, e.context.meta, r)
  }));
}

function noop() {}

function applyDefinitions(r, n, a) {
  for (var o = 0; o < a.length; o++) {
    if (a[o].kind === e.Kind.FRAGMENT_DEFINITION) {
      var i = a[o].name.value;
      var u = t.stringifyDocument(a[o]);
      if (!r.has(i)) {
        r.set(i, u);
        n.push(a[o]);
      } else if ("production" !== process.env.NODE_ENV && r.get(i) !== u) {
        console.warn("[WARNING: Duplicate Fragment] A fragment with name `" + i + "` already exists in this document.\nWhile fragment names may not be unique across your source, each name must be unique per document.");
      }
    } else {
      n.push(a[o]);
    }
  }
}

function shouldSkip(e) {
  var t = e.kind;
  return "mutation" !== t && "query" !== t;
}

function cacheExchange(e) {
  var n = e.forward;
  var a = e.client;
  var o = e.dispatchDebug;
  var i = new Map;
  var u = Object.create(null);
  function mapTypeNames(e) {
    var t = makeOperation(e.kind, e);
    t.query = formatDocument(e.query);
    return t;
  }
  function isOperationCached(e) {
    var t = e.context.requestPolicy;
    return "query" === e.kind && "network-only" !== t && ("cache-only" === t || i.has(e.key));
  }
  return function(e) {
    var c = r.share(e);
    var s = r.map((function(e) {
      var r = i.get(e.key);
      "production" !== process.env.NODE_ENV && o(t._extends({}, {
        operation: e
      }, r ? {
        type: "cacheHit",
        message: "The result was successfully retried from the cache"
      } : {
        type: "cacheMiss",
        message: "The result could not be retrieved from the cache"
      }));
      var n = t._extends({}, r, {
        operation: addMetadata(e, {
          cacheOutcome: r ? "hit" : "miss"
        })
      });
      if ("cache-and-network" === e.context.requestPolicy) {
        n.stale = !0;
        reexecuteOperation(a, e);
      }
      return n;
    }))(r.filter((function(e) {
      return !shouldSkip(e) && isOperationCached(e);
    }))(c));
    var p = r.tap((function(e) {
      var t = e.operation;
      if (!t) {
        return;
      }
      var r = collectTypesFromResponse(e.data).concat(t.context.additionalTypenames || []);
      if ("mutation" === e.operation.kind) {
        var n = new Set;
        "production" !== process.env.NODE_ENV && o({
          type: "cacheInvalidation",
          message: "The following typenames have been invalidated: " + r,
          operation: t,
          data: {
            typenames: r,
            response: e
          },
          source: "cacheExchange"
        });
        for (var c = 0; c < r.length; c++) {
          var s = r[c];
          var p = u[s] || (u[s] = new Set);
          p.forEach((function(e) {
            n.add(e);
          }));
          p.clear();
        }
        n.forEach((function(e) {
          if (i.has(e)) {
            t = i.get(e).operation;
            i.delete(e);
            reexecuteOperation(a, t);
          }
        }));
      } else if ("query" === t.kind && e.data) {
        i.set(t.key, e);
        for (var f = 0; f < r.length; f++) {
          var l = r[f];
          (u[l] || (u[l] = new Set)).add(t.key);
        }
      }
    }))(n(r.filter((function(e) {
      return "query" !== e.kind || "cache-only" !== e.context.requestPolicy;
    }))(r.map((function(e) {
      return addMetadata(e, {
        cacheOutcome: "miss"
      });
    }))(r.merge([ r.map(mapTypeNames)(r.filter((function(e) {
      return !shouldSkip(e) && !isOperationCached(e);
    }))(c)), r.filter((function(e) {
      return shouldSkip(e);
    }))(c) ])))));
    return r.merge([ s, p ]);
  };
}

function reexecuteOperation(e, r) {
  return e.reexecuteOperation(makeOperation(r.kind, r, t._extends({}, r.context, {
    requestPolicy: "network-only"
  })));
}

var a = new Set;

function dedupExchange(e) {
  var t = e.forward;
  var n = e.dispatchDebug;
  var a = new Set;
  function filterIncomingOperation(e) {
    var t = e.key;
    var r = e.kind;
    if ("teardown" === r) {
      a.delete(t);
      return !0;
    }
    if ("query" !== r && "subscription" !== r) {
      return !0;
    }
    var o = a.has(t);
    a.add(t);
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
      a.delete(e.operation.key);
    }
  }
  return function(e) {
    var n = r.filter(filterIncomingOperation)(e);
    return r.tap(afterOperationResult)(t(n));
  };
}

function fetchExchange(e) {
  var n = e.forward;
  var a = e.dispatchDebug;
  return function(e) {
    var o = r.share(e);
    var i = r.mergeMap((function(e) {
      var n = e.key;
      var i = r.filter((function(e) {
        return "teardown" === e.kind && e.key === n;
      }))(o);
      var u = t.makeFetchBody(e);
      var c = t.makeFetchURL(e, u);
      var s = t.makeFetchOptions(e, u);
      "production" !== process.env.NODE_ENV && a({
        type: "fetchRequest",
        message: "A fetch request is being executed.",
        operation: e,
        data: {
          url: c,
          fetchOptions: s
        },
        source: "fetchExchange"
      });
      return r.onPush((function(t) {
        var r = !t.data ? t.error : void 0;
        "production" !== process.env.NODE_ENV && a({
          type: r ? "fetchError" : "fetchSuccess",
          message: "A " + (r ? "failed" : "successful") + " fetch response has been returned.",
          operation: e,
          data: {
            url: c,
            fetchOptions: s,
            value: r || t
          },
          source: "fetchExchange"
        });
      }))(r.takeUntil(i)(t.makeFetchSource(e, c, s)));
    }))(r.filter((function(e) {
      return "query" === e.kind || "mutation" === e.kind;
    }))(o));
    var u = n(r.filter((function(e) {
      return "query" !== e.kind && "mutation" !== e.kind;
    }))(o));
    return r.merge([ i, u ]);
  };
}

function fallbackExchange(e) {
  var t = e.dispatchDebug;
  return function(e) {
    return r.filter((function() {
      return !1;
    }))(r.tap((function(e) {
      if ("teardown" !== e.kind && "production" !== process.env.NODE_ENV) {
        var r = 'No exchange has handled operations of kind "' + e.kind + "\". Check whether you've added an exchange responsible for these operations.";
        "production" !== process.env.NODE_ENV && t({
          type: "fallbackCatch",
          message: r,
          operation: e,
          source: "fallbackExchange"
        });
        console.warn(r);
      }
    }))(e));
  };
}

var o = fallbackExchange({
  dispatchDebug: noop
});

function composeExchanges(e) {
  return function(r) {
    var n = r.client;
    var a = r.dispatchDebug;
    return e.reduceRight((function(e, r) {
      return r({
        client: n,
        forward: e,
        dispatchDebug: function dispatchDebug$1(e) {
          "production" !== process.env.NODE_ENV && a(t._extends({}, {
            timestamp: Date.now(),
            source: r.name
          }, e));
        }
      });
    }), r.forward);
  };
}

var i = [ dedupExchange, cacheExchange, fetchExchange ];

var u = function Client(e) {
  if ("production" !== process.env.NODE_ENV && !e.url) {
    throw new Error("You are creating an urql-client without a url.");
  }
  var n = new Map;
  var a = new Map;
  var o = [];
  var u = r.makeSubject();
  var c = u.source;
  var s = u.next;
  var p = !1;
  function dispatchOperation(e) {
    p = !0;
    if (e) {
      s(e);
    }
    while (e = o.shift()) {
      s(e);
    }
    p = !1;
  }
  function makeResultSource(e) {
    var i = r.filter((function(t) {
      return t.operation.kind === e.kind && t.operation.key === e.key;
    }))(k);
    if (l.maskTypename) {
      i = r.map((function(e) {
        return t._extends({}, e, {
          data: maskTypename(e.data)
        });
      }))(i);
    }
    if ("mutation" === e.kind) {
      return r.take(1)(r.onStart((function() {
        return dispatchOperation(e);
      }))(i));
    }
    return r.share(r.onEnd((function() {
      n.delete(e.key);
      a.delete(e.key);
      for (var t = o.length - 1; t >= 0; t--) {
        if (o[t].key === e.key) {
          o.splice(t, 1);
        }
      }
      dispatchOperation(makeOperation("teardown", e, e.context));
    }))(r.onPush((function(t) {
      n.set(e.key, t);
    }))(r.switchMap((function(n) {
      if ("query" !== e.kind || n.stale) {
        return r.fromValue(n);
      }
      return r.merge([ r.fromValue(n), r.map((function() {
        return t._extends({}, n, {
          stale: !0
        });
      }))(r.take(1)(r.filter((function(t) {
        return "query" === t.kind && t.key === e.key && "cache-only" !== t.context.requestPolicy;
      }))(c))) ]);
    }))(r.takeUntil(r.filter((function(t) {
      return "teardown" === t.kind && t.key === e.key;
    }))(c))(i)))));
  }
  var f = this instanceof Client ? this : Object.create(Client.prototype);
  var l = t._extends(f, {
    url: e.url,
    fetchOptions: e.fetchOptions,
    fetch: e.fetch,
    suspense: !!e.suspense,
    requestPolicy: e.requestPolicy || "cache-first",
    preferGetMethod: !!e.preferGetMethod,
    maskTypename: !!e.maskTypename,
    operations$: c,
    reexecuteOperation: function reexecuteOperation(e) {
      if ("mutation" === e.kind || a.has(e.key)) {
        o.push(e);
        if (!p) {
          Promise.resolve().then(dispatchOperation);
        }
      }
    },
    createOperationContext: function createOperationContext(e) {
      if (!e) {
        e = {};
      }
      return t._extends({}, {
        url: l.url,
        fetchOptions: l.fetchOptions,
        fetch: l.fetch,
        preferGetMethod: l.preferGetMethod
      }, e, {
        suspense: e.suspense || !1 !== e.suspense && l.suspense,
        requestPolicy: e.requestPolicy || l.requestPolicy
      });
    },
    createRequestOperation: function createRequestOperation(e, r, n) {
      var a = t.getOperationType(r.query);
      if ("production" !== process.env.NODE_ENV && "teardown" !== e && a !== e) {
        throw new Error('Expected operation of type "' + e + '" but found "' + a + '"');
      }
      return makeOperation(e, r, l.createOperationContext(n));
    },
    executeRequestOperation: function executeRequestOperation(e) {
      if ("mutation" === e.kind) {
        return makeResultSource(e);
      }
      return r.make((function(o) {
        var i = a.get(e.key);
        if (!i) {
          a.set(e.key, i = makeResultSource(e));
        }
        var u = "cache-and-network" === e.context.requestPolicy || "network-only" === e.context.requestPolicy;
        return r.subscribe(o.next)(r.onEnd(o.complete)(r.onStart((function() {
          var r = n.get(e.key);
          if ("subscription" === e.kind) {
            return dispatchOperation(e);
          } else if (u) {
            dispatchOperation(e);
          }
          if (null != r && r === n.get(e.key)) {
            o.next(u ? t._extends({}, r, {
              stale: !0
            }) : r);
          } else if (!u) {
            dispatchOperation(e);
          }
        }))(i))).unsubscribe;
      }));
    },
    executeQuery: function executeQuery(e, t) {
      var r = l.createRequestOperation("query", e, t);
      return l.executeRequestOperation(r);
    },
    executeSubscription: function executeSubscription(e, t) {
      var r = l.createRequestOperation("subscription", e, t);
      return l.executeRequestOperation(r);
    },
    executeMutation: function executeMutation(e, t) {
      var r = l.createRequestOperation("mutation", e, t);
      return l.executeRequestOperation(r);
    },
    query: function query(e, r, n) {
      if (!n || "boolean" != typeof n.suspense) {
        n = t._extends({}, n, {
          suspense: !1
        });
      }
      return withPromise(l.executeQuery(t.createRequest(e, r), n));
    },
    readQuery: function readQuery(e, t, n) {
      var a = null;
      r.subscribe((function(e) {
        a = e;
      }))(l.query(e, t, n)).unsubscribe();
      return a;
    },
    subscription: function subscription(e, r, n) {
      return l.executeSubscription(t.createRequest(e, r), n);
    },
    mutation: function mutation(e, r, n) {
      return withPromise(l.executeMutation(t.createRequest(e, r), n));
    }
  });
  var d = noop;
  if ("production" !== process.env.NODE_ENV) {
    var h = r.makeSubject();
    var v = h.next;
    var m = h.source;
    l.subscribeToDebugTarget = function(e) {
      return r.subscribe(e)(m);
    };
    d = v;
  }
  var y = composeExchanges(void 0 !== e.exchanges ? e.exchanges : i);
  var k = r.share(y({
    client: l,
    dispatchDebug: d,
    forward: fallbackExchange({
      dispatchDebug: d
    })
  })(c));
  r.publish(k);
  return l;
};

var c = u;

exports.CombinedError = t.CombinedError;

exports.createRequest = t.createRequest;

exports.getOperationName = t.getOperationName;

exports.makeErrorResult = t.makeErrorResult;

exports.makeResult = t.makeResult;

exports.mergeResultPatch = t.mergeResultPatch;

exports.stringifyVariables = t.stringifyVariables;

exports.Client = u;

exports.cacheExchange = cacheExchange;

exports.composeExchanges = composeExchanges;

exports.createClient = c;

exports.debugExchange = function debugExchange(e) {
  var t = e.forward;
  if ("production" === process.env.NODE_ENV) {
    return function(e) {
      return t(e);
    };
  } else {
    return function(e) {
      return r.tap((function(e) {
        return console.log("[Exchange debug]: Completed operation: ", e);
      }))(t(r.tap((function(e) {
        return console.log("[Exchange debug]: Incoming operation: ", e);
      }))(e)));
    };
  }
};

exports.dedupExchange = dedupExchange;

exports.defaultExchanges = i;

exports.errorExchange = function errorExchange(e) {
  var t = e.onError;
  return function(e) {
    var n = e.forward;
    return function(e) {
      return r.tap((function(e) {
        var r = e.error;
        if (r) {
          t(r, e.operation);
        }
      }))(n(e));
    };
  };
};

exports.fallbackExchangeIO = o;

exports.fetchExchange = fetchExchange;

exports.formatDocument = formatDocument;

exports.gql = function gql() {
  var r = arguments;
  var n = new Map;
  var a = [];
  var o = [];
  var i = Array.isArray(arguments[0]) ? arguments[0][0] : arguments[0] || "";
  for (var u = 1; u < arguments.length; u++) {
    var c = r[u];
    if (c && c.definitions) {
      o.push.apply(o, c.definitions);
    } else {
      i += c;
    }
    i += r[0][u];
  }
  applyDefinitions(n, a, t.keyDocument(i).definitions);
  applyDefinitions(n, a, o);
  return t.keyDocument({
    kind: e.Kind.DOCUMENT,
    definitions: a
  });
};

exports.makeOperation = makeOperation;

exports.maskTypename = maskTypename;

exports.ssrExchange = function ssrExchange(e) {
  var n = !(!e || !e.staleWhileRevalidate);
  var o = !(!e || !e.includeExtensions);
  var i = {};
  var u = [];
  function invalidate(e) {
    u.push(e.operation.key);
    if (1 === u.length) {
      Promise.resolve().then((function() {
        var e;
        while (e = u.shift()) {
          i[e] = null;
        }
      }));
    }
  }
  var ssr = function(u) {
    var c = u.client;
    var s = u.forward;
    return function(u) {
      var p = e && "boolean" == typeof e.isClient ? !!e.isClient : !c.suspense;
      var f = r.share(u);
      var l = s(r.filter((function(e) {
        return !i[e.key] || !!i[e.key].hasNext;
      }))(f));
      var d = r.map((function(e) {
        var r = function deserializeResult(e, r, n) {
          return {
            operation: e,
            data: r.data ? JSON.parse(r.data) : void 0,
            extensions: n && r.extensions ? JSON.parse(r.extensions) : void 0,
            error: r.error ? new t.CombinedError({
              networkError: r.error.networkError ? new Error(r.error.networkError) : void 0,
              graphQLErrors: r.error.graphQLErrors
            }) : void 0,
            hasNext: r.hasNext
          };
        }(e, i[e.key], o);
        if (n && !a.has(e.key)) {
          r.stale = !0;
          a.add(e.key);
          reexecuteOperation(c, e);
        }
        return r;
      }))(r.filter((function(e) {
        return !!i[e.key];
      }))(f));
      if (!p) {
        l = r.tap((function(e) {
          var t = e.operation;
          if ("mutation" !== t.kind) {
            var r = function serializeResult(e, t) {
              var r = e.hasNext;
              var n = e.data;
              var a = e.extensions;
              var o = e.error;
              var i = {};
              if (void 0 !== n) {
                i.data = JSON.stringify(n);
              }
              if (t && void 0 !== a) {
                i.extensions = JSON.stringify(a);
              }
              if (r) {
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
            }(e, o);
            i[t.key] = r;
          }
        }))(l);
      } else {
        d = r.tap(invalidate)(d);
      }
      return r.merge([ l, d ]);
    };
  };
  ssr.restoreData = function(e) {
    for (var t in e) {
      if (null !== i[t]) {
        i[t] = e[t];
      }
    }
  };
  ssr.extractData = function() {
    var e = {};
    for (var t in i) {
      if (null != i[t]) {
        e[t] = i[t];
      }
    }
    return e;
  };
  if (e && e.initialState) {
    ssr.restoreData(e.initialState);
  }
  return ssr;
};

exports.subscriptionExchange = function subscriptionExchange(n) {
  var a = n.forwardSubscription;
  var o = n.enableAllOperations;
  return function(n) {
    var i = n.client;
    var u = n.forward;
    function isSubscriptionOperation(e) {
      var t = e.kind;
      return "subscription" === t || !!o && ("query" === t || "mutation" === t);
    }
    return function(n) {
      var o = r.share(n);
      var c = r.mergeMap((function(n) {
        var u = n.key;
        var c = r.filter((function(e) {
          return "teardown" === e.kind && e.key === u;
        }))(o);
        return r.takeUntil(c)(function createSubscriptionSource(n) {
          var o = a({
            key: n.key.toString(36),
            query: e.print(n.query),
            variables: n.variables,
            context: t._extends({}, n.context)
          });
          return r.make((function(e) {
            var r = e.next;
            var a = e.complete;
            var u = !1;
            var c;
            Promise.resolve().then((function() {
              if (u) {
                return;
              }
              c = o.subscribe({
                next: function(e) {
                  return r(t.makeResult(n, e));
                },
                error: function(e) {
                  return r(t.makeErrorResult(n, e));
                },
                complete: function() {
                  if (!u) {
                    u = !0;
                    if ("subscription" === n.kind) {
                      i.reexecuteOperation(makeOperation("teardown", n, n.context));
                    }
                    a();
                  }
                }
              });
            }));
            return function() {
              u = !0;
              if (c) {
                c.unsubscribe();
              }
            };
          }));
        }(n));
      }))(r.filter(isSubscriptionOperation)(o));
      var s = u(r.filter((function(e) {
        return !isSubscriptionOperation(e);
      }))(o));
      return r.merge([ c, s ]);
    };
  };
};
//# sourceMappingURL=urql-core.js.map
