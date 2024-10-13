var e = require("graphql");

var r = require("wonka");

function rehydrateGraphQlError(r) {
  if ("string" == typeof r) {
    return new e.GraphQLError(r);
  } else if ("object" == typeof r && r.message) {
    return new e.GraphQLError(r.message, r.nodes, r.source, r.positions, r.path, r, r.extensions || {});
  } else {
    return r;
  }
}

var t = function(e) {
  function CombinedError(r) {
    var t = r.networkError;
    var n = r.response;
    var o = (r.graphQLErrors || []).map(rehydrateGraphQlError);
    var a = function generateErrorMessage(e, r) {
      var t = "";
      if (void 0 !== e) {
        return t = "[Network] " + e.message;
      }
      if (void 0 !== r) {
        r.forEach((function(e) {
          t += "[GraphQL] " + e.message + "\n";
        }));
      }
      return t.trim();
    }(t, o);
    e.call(this, a);
    this.name = "CombinedError";
    this.message = a;
    this.graphQLErrors = o;
    this.networkError = t;
    this.response = n;
  }
  if (e) {
    CombinedError.__proto__ = e;
  }
  (CombinedError.prototype = Object.create(e && e.prototype)).constructor = CombinedError;
  CombinedError.prototype.toString = function toString() {
    return this.message;
  };
  return CombinedError;
}(Error);

function phash(e, r) {
  e |= 0;
  for (var t = 0, n = 0 | r.length; t < n; t++) {
    e = (e << 5) + e + r.charCodeAt(t);
  }
  return e;
}

function hash(e) {
  return phash(5381, e) >>> 0;
}

var n = new Set;

var o = new WeakMap;

function stringify(e) {
  if (null === e || n.has(e)) {
    return "null";
  } else if ("object" != typeof e) {
    return JSON.stringify(e) || "";
  } else if (e.toJSON) {
    return stringify(e.toJSON());
  } else if (Array.isArray(e)) {
    var r = "[";
    for (var t = 0, a = e.length; t < a; t++) {
      if (t > 0) {
        r += ",";
      }
      var i = stringify(e[t]);
      r += i.length > 0 ? i : "null";
    }
    return r += "]";
  }
  var s = Object.keys(e).sort();
  if (!s.length && e.constructor && e.constructor !== Object) {
    var u = o.get(e) || Math.random().toString(36).slice(2);
    o.set(e, u);
    return '{"__key":"' + u + '"}';
  }
  n.add(e);
  var f = "{";
  for (var c = 0, l = s.length; c < l; c++) {
    var p = s[c];
    var h = stringify(e[p]);
    if (h) {
      if (f.length > 1) {
        f += ",";
      }
      f += stringify(p) + ":" + h;
    }
  }
  n.delete(e);
  return f += "}";
}

function stringifyVariables(e) {
  n.clear();
  return stringify(e);
}

function stringifyDocument(r) {
  var t = ("string" != typeof r ? r.loc && r.loc.source.body || e.print(r) : r).replace(/([\s,]|#[^\n\r]+)+/g, " ").trim();
  if ("string" != typeof r) {
    var n = "definitions" in r && getOperationName(r);
    if (n) {
      t = "# " + n + "\n" + t;
    }
    if (!r.loc) {
      r.loc = {
        start: 0,
        end: t.length,
        source: {
          body: t,
          name: "gql",
          locationOffset: {
            line: 1,
            column: 1
          }
        }
      };
    }
  }
  return t;
}

var a = new Map;

function keyDocument(r) {
  var t;
  var n;
  if ("string" == typeof r) {
    t = hash(stringifyDocument(r));
    n = a.get(t) || e.parse(r, {
      noLocation: !0
    });
  } else {
    t = r.__key || hash(stringifyDocument(r));
    n = a.get(t) || r;
  }
  if (!n.loc) {
    stringifyDocument(n);
  }
  n.__key = t;
  a.set(t, n);
  return n;
}

function getOperationName(r) {
  for (var t = 0, n = r.definitions.length; t < n; t++) {
    var o = r.definitions[t];
    if (o.kind === e.Kind.OPERATION_DEFINITION && o.name) {
      return o.name.value;
    }
  }
}

function _extends() {
  return (_extends = Object.assign || function(e) {
    for (var r = 1; r < arguments.length; r++) {
      var t = arguments[r];
      for (var n in t) {
        if (Object.prototype.hasOwnProperty.call(t, n)) {
          e[n] = t[n];
        }
      }
    }
    return e;
  }).apply(this, arguments);
}

function makeResult(e, r, n) {
  if (!("data" in r) && !("errors" in r) || "path" in r) {
    throw new Error("No Content");
  }
  return {
    operation: e,
    data: r.data,
    error: Array.isArray(r.errors) ? new t({
      graphQLErrors: r.errors,
      response: n
    }) : void 0,
    extensions: "object" == typeof r.extensions && r.extensions || void 0,
    hasNext: !!r.hasNext
  };
}

function mergeResultPatch(e, r, n) {
  var o = _extends({}, e);
  o.hasNext = !!r.hasNext;
  if (!("path" in r)) {
    if ("data" in r) {
      o.data = r.data;
    }
    return o;
  }
  if (Array.isArray(r.errors)) {
    o.error = new t({
      graphQLErrors: o.error ? o.error.graphQLErrors.concat(r.errors) : r.errors,
      response: n
    });
  }
  var a = o.data = _extends({}, o.data);
  var i = 0;
  var s;
  while (i < r.path.length) {
    a = a[s = r.path[i++]] = Array.isArray(a[s]) ? [].concat(a[s]) : _extends({}, a[s]);
  }
  _extends(a, r.data);
  return o;
}

function makeErrorResult(e, r, n) {
  return {
    operation: e,
    data: void 0,
    error: new t({
      networkError: r,
      response: n
    }),
    extensions: void 0
  };
}

function shouldUseGet(e) {
  return "query" === e.kind && !!e.context.preferGetMethod;
}

var i = "undefined" != typeof Symbol ? Symbol.asyncIterator : null;

var s = "undefined" != typeof TextDecoder ? new TextDecoder : null;

var u = /content-type:[^\r\n]*application\/json/i;

var f = /boundary="?([^=";]+)"?/i;

function executeIncrementalFetch(e, r, t) {
  var n = t.headers && t.headers.get("Content-Type") || "";
  if (!/multipart\/mixed/i.test(n)) {
    return t.json().then((function(n) {
      e(makeResult(r, n, t));
    }));
  }
  var o = "---";
  var a = n.match(f);
  if (a) {
    o = "--" + a[1];
  }
  var c;
  var cancel = function() {};
  if (i && t[i]) {
    var l = t[i]();
    c = l.next.bind(l);
  } else if ("body" in t && t.body) {
    var p = t.body.getReader();
    cancel = p.cancel.bind(p);
    c = p.read.bind(p);
  } else {
    throw new TypeError("Streaming requests unsupported");
  }
  var h = "";
  var d = !0;
  var v = null;
  var y = null;
  return c().then((function next(n) {
    if (!n.done) {
      var a = function toString(e) {
        return "Buffer" === e.constructor.name ? e.toString() : s.decode(e);
      }(n.value);
      var i = a.indexOf(o);
      if (i > -1) {
        i += h.length;
      } else {
        i = h.indexOf(o);
      }
      h += a;
      while (i > -1) {
        var f = h.slice(0, i);
        var l = h.slice(i + o.length);
        if (d) {
          d = !1;
        } else {
          var p = f.indexOf("\r\n\r\n") + 4;
          var m = f.slice(0, p);
          var g = f.slice(p, f.lastIndexOf("\r\n"));
          var x = void 0;
          if (u.test(m)) {
            try {
              x = JSON.parse(g);
              v = y = y ? mergeResultPatch(y, x, t) : makeResult(r, x, t);
            } catch (e) {}
          }
          if ("--" === l.slice(0, 2) || x && !x.hasNext) {
            if (!y) {
              return e(makeResult(r, {}, t));
            }
            break;
          }
        }
        i = (h = l).indexOf(o);
      }
    }
    if (v) {
      e(v);
      v = null;
    }
    if (!n.done && (!y || y.hasNext)) {
      return c().then(next);
    }
  })).finally(cancel);
}

exports.CombinedError = t;

exports._extends = _extends;

exports.createRequest = function createRequest(e, r) {
  if (!r) {
    r = {};
  }
  var t = keyDocument(e);
  return {
    key: phash(t.__key, stringifyVariables(r)) >>> 0,
    query: t,
    variables: r
  };
};

exports.getOperationName = getOperationName;

exports.getOperationType = function getOperationType(r) {
  for (var t = 0, n = r.definitions.length; t < n; t++) {
    var o = r.definitions[t];
    if (o.kind === e.Kind.OPERATION_DEFINITION) {
      return o.operation;
    }
  }
};

exports.keyDocument = keyDocument;

exports.makeErrorResult = makeErrorResult;

exports.makeFetchBody = function makeFetchBody(r) {
  return {
    query: e.print(r.query),
    operationName: getOperationName(r.query),
    variables: r.variables || void 0,
    extensions: void 0
  };
};

exports.makeFetchOptions = function makeFetchOptions(e, r) {
  var t = shouldUseGet(e);
  var n = "function" == typeof e.context.fetchOptions ? e.context.fetchOptions() : e.context.fetchOptions || {};
  return _extends({}, n, {
    body: !t && r ? JSON.stringify(r) : void 0,
    method: t ? "GET" : "POST",
    headers: t ? n.headers : _extends({}, {
      "content-type": "application/json"
    }, n.headers)
  });
};

exports.makeFetchSource = function makeFetchSource(e, t, n) {
  var o = "manual" === n.redirect ? 400 : 300;
  var a = e.context.fetch;
  return r.make((function(r) {
    var i = r.next;
    var s = r.complete;
    var u = "undefined" != typeof AbortController ? new AbortController : null;
    if (u) {
      n.signal = u.signal;
    }
    var f = !1;
    var c = !1;
    var l;
    Promise.resolve().then((function() {
      if (f) {
        return;
      }
      return (a || fetch)(t, n);
    })).then((function(r) {
      if (!r) {
        return;
      }
      c = (l = r).status < 200 || l.status >= o;
      return executeIncrementalFetch(i, e, l);
    })).then(s).catch((function(r) {
      if ("AbortError" !== r.name) {
        var t = makeErrorResult(e, c ? new Error(l.statusText) : r, l);
        i(t);
        s();
      }
    }));
    return function() {
      f = !0;
      if (u) {
        u.abort();
      }
    };
  }));
};

exports.makeFetchURL = function makeFetchURL(e, r) {
  var t = shouldUseGet(e);
  var n = e.context.url;
  if (!t || !r) {
    return n;
  }
  var o = [];
  if (r.operationName) {
    o.push("operationName=" + encodeURIComponent(r.operationName));
  }
  if (r.query) {
    o.push("query=" + encodeURIComponent(r.query.replace(/#[^\n\r]+/g, " ").trim()));
  }
  if (r.variables) {
    o.push("variables=" + encodeURIComponent(stringifyVariables(r.variables)));
  }
  if (r.extensions) {
    o.push("extensions=" + encodeURIComponent(stringifyVariables(r.extensions)));
  }
  return n + "?" + o.join("&");
};

exports.makeResult = makeResult;

exports.mergeResultPatch = mergeResultPatch;

exports.stringifyDocument = stringifyDocument;

exports.stringifyVariables = stringifyVariables;
//# sourceMappingURL=878a8849.js.map
