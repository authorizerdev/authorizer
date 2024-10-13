import { GraphQLError as e } from "graphql/error/GraphQLError.mjs";

import { Kind as r } from "graphql/language/kinds.mjs";

import { parse as t } from "graphql/language/parser.mjs";

import { print as n } from "graphql/language/printer.mjs";

import { make as o } from "wonka";

function rehydrateGraphQlError(r) {
  if ("string" == typeof r) {
    return new e(r);
  } else if ("object" == typeof r && r.message) {
    return new e(r.message, r.nodes, r.source, r.positions, r.path, r, r.extensions || {});
  } else {
    return r;
  }
}

var a = function(e) {
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

var i = new Set;

var s = new WeakMap;

function stringify(e) {
  if (null === e || i.has(e)) {
    return "null";
  } else if ("object" != typeof e) {
    return JSON.stringify(e) || "";
  } else if (e.toJSON) {
    return stringify(e.toJSON());
  } else if (Array.isArray(e)) {
    var r = "[";
    for (var t = 0, n = e.length; t < n; t++) {
      if (t > 0) {
        r += ",";
      }
      var o = stringify(e[t]);
      r += o.length > 0 ? o : "null";
    }
    return r += "]";
  }
  var a = Object.keys(e).sort();
  if (!a.length && e.constructor && e.constructor !== Object) {
    var u = s.get(e) || Math.random().toString(36).slice(2);
    s.set(e, u);
    return '{"__key":"' + u + '"}';
  }
  i.add(e);
  var f = "{";
  for (var c = 0, l = a.length; c < l; c++) {
    var h = a[c];
    var p = stringify(e[h]);
    if (p) {
      if (f.length > 1) {
        f += ",";
      }
      f += stringify(h) + ":" + p;
    }
  }
  i.delete(e);
  return f += "}";
}

function stringifyVariables(e) {
  i.clear();
  return stringify(e);
}

function stringifyDocument(e) {
  var r = ("string" != typeof e ? e.loc && e.loc.source.body || n(e) : e).replace(/([\s,]|#[^\n\r]+)+/g, " ").trim();
  if ("string" != typeof e) {
    var t = "definitions" in e && getOperationName(e);
    if (t) {
      r = "# " + t + "\n" + r;
    }
    if (!e.loc) {
      e.loc = {
        start: 0,
        end: r.length,
        source: {
          body: r,
          name: "gql",
          locationOffset: {
            line: 1,
            column: 1
          }
        }
      };
    }
  }
  return r;
}

var u = new Map;

function keyDocument(e) {
  var r;
  var n;
  if ("string" == typeof e) {
    r = hash(stringifyDocument(e));
    n = u.get(r) || t(e, {
      noLocation: !0
    });
  } else {
    r = e.__key || hash(stringifyDocument(e));
    n = u.get(r) || e;
  }
  if (!n.loc) {
    stringifyDocument(n);
  }
  n.__key = r;
  u.set(r, n);
  return n;
}

function createRequest(e, r) {
  if (!r) {
    r = {};
  }
  var t = keyDocument(e);
  return {
    key: phash(t.__key, stringifyVariables(r)) >>> 0,
    query: t,
    variables: r
  };
}

function getOperationName(e) {
  for (var t = 0, n = e.definitions.length; t < n; t++) {
    var o = e.definitions[t];
    if (o.kind === r.OPERATION_DEFINITION && o.name) {
      return o.name.value;
    }
  }
}

function getOperationType(e) {
  for (var t = 0, n = e.definitions.length; t < n; t++) {
    var o = e.definitions[t];
    if (o.kind === r.OPERATION_DEFINITION) {
      return o.operation;
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

function makeResult(e, r, t) {
  if (!("data" in r) && !("errors" in r) || "path" in r) {
    throw new Error("No Content");
  }
  return {
    operation: e,
    data: r.data,
    error: Array.isArray(r.errors) ? new a({
      graphQLErrors: r.errors,
      response: t
    }) : void 0,
    extensions: "object" == typeof r.extensions && r.extensions || void 0,
    hasNext: !!r.hasNext
  };
}

function mergeResultPatch(e, r, t) {
  var n = _extends({}, e);
  n.hasNext = !!r.hasNext;
  if (!("path" in r)) {
    if ("data" in r) {
      n.data = r.data;
    }
    return n;
  }
  if (Array.isArray(r.errors)) {
    n.error = new a({
      graphQLErrors: n.error ? n.error.graphQLErrors.concat(r.errors) : r.errors,
      response: t
    });
  }
  var o = n.data = _extends({}, n.data);
  var i = 0;
  var s;
  while (i < r.path.length) {
    o = o[s = r.path[i++]] = Array.isArray(o[s]) ? [].concat(o[s]) : _extends({}, o[s]);
  }
  _extends(o, r.data);
  return n;
}

function makeErrorResult(e, r, t) {
  return {
    operation: e,
    data: void 0,
    error: new a({
      networkError: r,
      response: t
    }),
    extensions: void 0
  };
}

function shouldUseGet(e) {
  return "query" === e.kind && !!e.context.preferGetMethod;
}

function makeFetchBody(e) {
  return {
    query: n(e.query),
    operationName: getOperationName(e.query),
    variables: e.variables || void 0,
    extensions: void 0
  };
}

function makeFetchURL(e, r) {
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
}

function makeFetchOptions(e, r) {
  var t = shouldUseGet(e);
  var n = "function" == typeof e.context.fetchOptions ? e.context.fetchOptions() : e.context.fetchOptions || {};
  return _extends({}, n, {
    body: !t && r ? JSON.stringify(r) : void 0,
    method: t ? "GET" : "POST",
    headers: t ? n.headers : _extends({}, {
      "content-type": "application/json"
    }, n.headers)
  });
}

var f = "undefined" != typeof Symbol ? Symbol.asyncIterator : null;

var c = "undefined" != typeof TextDecoder ? new TextDecoder : null;

var l = /content-type:[^\r\n]*application\/json/i;

var h = /boundary="?([^=";]+)"?/i;

function executeIncrementalFetch(e, r, t) {
  var n = t.headers && t.headers.get("Content-Type") || "";
  if (!/multipart\/mixed/i.test(n)) {
    return t.json().then((function(n) {
      e(makeResult(r, n, t));
    }));
  }
  var o = "---";
  var a = n.match(h);
  if (a) {
    o = "--" + a[1];
  }
  var i;
  var cancel = function() {};
  if (f && t[f]) {
    var s = t[f]();
    i = s.next.bind(s);
  } else if ("body" in t && t.body) {
    var u = t.body.getReader();
    cancel = u.cancel.bind(u);
    i = u.read.bind(u);
  } else {
    throw new TypeError("Streaming requests unsupported");
  }
  var p = "";
  var d = !0;
  var m = null;
  var v = null;
  return i().then((function next(n) {
    if (!n.done) {
      var a = function toString(e) {
        return "Buffer" === e.constructor.name ? e.toString() : c.decode(e);
      }(n.value);
      var s = a.indexOf(o);
      if (s > -1) {
        s += p.length;
      } else {
        s = p.indexOf(o);
      }
      p += a;
      while (s > -1) {
        var u = p.slice(0, s);
        var f = p.slice(s + o.length);
        if (d) {
          d = !1;
        } else {
          var h = u.indexOf("\r\n\r\n") + 4;
          var g = u.slice(0, h);
          var y = u.slice(h, u.lastIndexOf("\r\n"));
          var x = void 0;
          if (l.test(g)) {
            try {
              x = JSON.parse(y);
              m = v = v ? mergeResultPatch(v, x, t) : makeResult(r, x, t);
            } catch (e) {}
          }
          if ("--" === f.slice(0, 2) || x && !x.hasNext) {
            if (!v) {
              return e(makeResult(r, {}, t));
            }
            break;
          }
        }
        s = (p = f).indexOf(o);
      }
    }
    if (m) {
      e(m);
      m = null;
    }
    if (!n.done && (!v || v.hasNext)) {
      return i().then(next);
    }
  })).finally(cancel);
}

function makeFetchSource(e, r, t) {
  var n = "manual" === t.redirect ? 400 : 300;
  var a = e.context.fetch;
  return o((function(o) {
    var i = o.next;
    var s = o.complete;
    var u = "undefined" != typeof AbortController ? new AbortController : null;
    if (u) {
      t.signal = u.signal;
    }
    var f = !1;
    var c = !1;
    var l;
    Promise.resolve().then((function() {
      if (f) {
        return;
      }
      return (a || fetch)(r, t);
    })).then((function(r) {
      if (!r) {
        return;
      }
      c = (l = r).status < 200 || l.status >= n;
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
}

export { a as C, _extends as _, makeErrorResult as a, makeFetchBody as b, makeFetchURL as c, makeFetchOptions as d, makeFetchSource as e, createRequest as f, getOperationType as g, stringifyVariables as h, mergeResultPatch as i, getOperationName as j, keyDocument as k, makeResult as m, stringifyDocument as s };
//# sourceMappingURL=01e1547d.mjs.map
