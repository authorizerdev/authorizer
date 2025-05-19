var teardownPlaceholder = () => {};

var e = teardownPlaceholder;

function start(e) {
  return {
    tag: 0,
    0: e
  };
}

function push(e) {
  return {
    tag: 1,
    0: e
  };
}

var asyncIteratorSymbol = () => "function" == typeof Symbol && Symbol.asyncIterator || "@@asyncIterator";

var observableSymbol = () => "function" == typeof Symbol && Symbol.observable || "@@observable";

var identity = e => e;

function buffer(r) {
  return t => i => {
    var a = [];
    var f = e;
    var n = e;
    var s = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        n(1);
        if (a.length) {
          i(push(a));
        }
        i(0);
      } else if (0 === e.tag) {
        f = e[0];
        r((e => {
          if (l) {} else if (0 === e) {
            l = !0;
            f(1);
            if (a.length) {
              i(push(a));
            }
            i(0);
          } else if (0 === e.tag) {
            n = e[0];
          } else if (a.length) {
            var r = push(a);
            a = [];
            i(r);
          }
        }));
      } else {
        a.push(e[0]);
        if (!s) {
          s = !0;
          f(0);
          n(0);
        } else {
          s = !1;
        }
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        f(1);
        n(1);
      } else if (!l && !s) {
        s = !0;
        f(0);
        n(0);
      }
    })));
  };
}

function concatMap(r) {
  return t => i => {
    var a = [];
    var f = e;
    var n = e;
    var s = !1;
    var l = !1;
    var u = !1;
    var o = !1;
    function applyInnerSource(e) {
      u = !0;
      e((e => {
        if (0 === e) {
          if (u) {
            u = !1;
            if (a.length) {
              applyInnerSource(r(a.shift()));
            } else if (o) {
              i(0);
            } else if (!s) {
              s = !0;
              f(0);
            }
          }
        } else if (0 === e.tag) {
          l = !1;
          (n = e[0])(0);
        } else if (u) {
          i(e);
          if (l) {
            l = !1;
          } else {
            n(0);
          }
        }
      }));
    }
    t((e => {
      if (o) {} else if (0 === e) {
        o = !0;
        if (!u && !a.length) {
          i(0);
        }
      } else if (0 === e.tag) {
        f = e[0];
      } else {
        s = !1;
        if (u) {
          a.push(e[0]);
        } else {
          applyInnerSource(r(e[0]));
        }
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!o) {
          o = !0;
          f(1);
        }
        if (u) {
          u = !1;
          n(1);
        }
      } else {
        if (!o && !s) {
          s = !0;
          f(0);
        }
        if (u && !l) {
          l = !0;
          n(0);
        }
      }
    })));
  };
}

function concatAll(e) {
  return concatMap(identity)(e);
}

function concat(e) {
  return concatAll(r(e));
}

function filter(r) {
  return t => i => {
    var a = e;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        i(e);
      } else if (!r(e[0])) {
        a(0);
      } else {
        i(e);
      }
    }));
  };
}

function map(e) {
  return r => t => r((r => {
    if (0 === r || 0 === r.tag) {
      t(r);
    } else {
      t(push(e(r[0])));
    }
  }));
}

function mergeMap(r) {
  return t => i => {
    var a = [];
    var f = e;
    var n = !1;
    var s = !1;
    t((t => {
      if (s) {} else if (0 === t) {
        s = !0;
        if (!a.length) {
          i(0);
        }
      } else if (0 === t.tag) {
        f = t[0];
      } else {
        n = !1;
        !function applyInnerSource(r) {
          var t = e;
          r((e => {
            if (0 === e) {
              if (a.length) {
                var r = a.indexOf(t);
                if (r > -1) {
                  (a = a.slice()).splice(r, 1);
                }
                if (!a.length) {
                  if (s) {
                    i(0);
                  } else if (!n) {
                    n = !0;
                    f(0);
                  }
                }
              }
            } else if (0 === e.tag) {
              a.push(t = e[0]);
              t(0);
            } else if (a.length) {
              i(e);
              t(0);
            }
          }));
        }(r(t[0]));
        if (!n) {
          n = !0;
          f(0);
        }
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!s) {
          s = !0;
          f(1);
        }
        for (var r = 0, t = a, i = a.length; r < i; r++) {
          t[r](1);
        }
        a.length = 0;
      } else {
        if (!s && !n) {
          n = !0;
          f(0);
        } else {
          n = !1;
        }
        for (var l = 0, u = a, o = a.length; l < o; l++) {
          u[l](0);
        }
      }
    })));
  };
}

function mergeAll(e) {
  return mergeMap(identity)(e);
}

function merge(e) {
  return mergeAll(r(e));
}

function onEnd(e) {
  return r => t => {
    var i = !1;
    r((r => {
      if (i) {} else if (0 === r) {
        i = !0;
        t(0);
        e();
      } else if (0 === r.tag) {
        var a = r[0];
        t(start((r => {
          if (1 === r) {
            i = !0;
            a(1);
            e();
          } else {
            a(r);
          }
        })));
      } else {
        t(r);
      }
    }));
  };
}

function onPush(e) {
  return r => t => {
    var i = !1;
    r((r => {
      if (i) {} else if (0 === r) {
        i = !0;
        t(0);
      } else if (0 === r.tag) {
        var a = r[0];
        t(start((e => {
          if (1 === e) {
            i = !0;
          }
          a(e);
        })));
      } else {
        e(r[0]);
        t(r);
      }
    }));
  };
}

function onStart(e) {
  return r => t => r((r => {
    if (0 === r) {
      t(0);
    } else if (0 === r.tag) {
      t(r);
      e();
    } else {
      t(r);
    }
  }));
}

function sample(r) {
  return t => i => {
    var a = e;
    var f = e;
    var n;
    var s = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        f(1);
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
      } else {
        n = e[0];
        if (!s) {
          s = !0;
          f(0);
          a(0);
        } else {
          s = !1;
        }
      }
    }));
    r((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        a(1);
        i(0);
      } else if (0 === e.tag) {
        f = e[0];
      } else if (void 0 !== n) {
        var r = push(n);
        n = void 0;
        i(r);
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        a(1);
        f(1);
      } else if (!l && !s) {
        s = !0;
        a(0);
        f(0);
      }
    })));
  };
}

function scan(e, r) {
  return t => i => {
    var a = r;
    t((r => {
      if (0 === r) {
        i(0);
      } else if (0 === r.tag) {
        i(r);
      } else {
        i(push(a = e(a, r[0])));
      }
    }));
  };
}

function share(r) {
  var t = [];
  var i = e;
  var a = !1;
  return e => {
    t.push(e);
    if (1 === t.length) {
      r((e => {
        if (0 === e) {
          for (var r = 0, f = t, n = t.length; r < n; r++) {
            f[r](0);
          }
          t.length = 0;
        } else if (0 === e.tag) {
          i = e[0];
        } else {
          a = !1;
          for (var s = 0, l = t, u = t.length; s < u; s++) {
            l[s](e);
          }
        }
      }));
    }
    e(start((r => {
      if (1 === r) {
        var f = t.indexOf(e);
        if (f > -1) {
          (t = t.slice()).splice(f, 1);
        }
        if (!t.length) {
          i(1);
        }
      } else if (!a) {
        a = !0;
        i(0);
      }
    })));
  };
}

function skip(r) {
  return t => i => {
    var a = e;
    var f = r;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        i(e);
      } else if (f-- > 0) {
        a(0);
      } else {
        i(e);
      }
    }));
  };
}

function skipUntil(r) {
  return t => i => {
    var a = e;
    var f = e;
    var n = !0;
    var s = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        if (n) {
          f(1);
        }
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        r((e => {
          if (0 === e) {
            if (n) {
              l = !0;
              a(1);
            }
          } else if (0 === e.tag) {
            (f = e[0])(0);
          } else {
            n = !1;
            f(1);
          }
        }));
      } else if (!n) {
        s = !1;
        i(e);
      } else if (!s) {
        s = !0;
        a(0);
        f(0);
      } else {
        s = !1;
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        a(1);
        if (n) {
          f(1);
        }
      } else if (!l && !s) {
        s = !0;
        if (n) {
          f(0);
        }
        a(0);
      }
    })));
  };
}

function skipWhile(r) {
  return t => i => {
    var a = e;
    var f = !0;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        i(e);
      } else if (f) {
        if (r(e[0])) {
          a(0);
        } else {
          f = !1;
          i(e);
        }
      } else {
        i(e);
      }
    }));
  };
}

function switchMap(r) {
  return t => i => {
    var a = e;
    var f = e;
    var n = !1;
    var s = !1;
    var l = !1;
    var u = !1;
    t((t => {
      if (u) {} else if (0 === t) {
        u = !0;
        if (!l) {
          i(0);
        }
      } else if (0 === t.tag) {
        a = t[0];
      } else {
        if (l) {
          f(1);
          f = e;
        }
        if (!n) {
          n = !0;
          a(0);
        } else {
          n = !1;
        }
        !function applyInnerSource(e) {
          l = !0;
          e((e => {
            if (!l) {} else if (0 === e) {
              l = !1;
              if (u) {
                i(0);
              } else if (!n) {
                n = !0;
                a(0);
              }
            } else if (0 === e.tag) {
              s = !1;
              (f = e[0])(0);
            } else {
              i(e);
              if (!s) {
                f(0);
              } else {
                s = !1;
              }
            }
          }));
        }(r(t[0]));
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!u) {
          u = !0;
          a(1);
        }
        if (l) {
          l = !1;
          f(1);
        }
      } else {
        if (!u && !n) {
          n = !0;
          a(0);
        }
        if (l && !s) {
          s = !0;
          f(0);
        }
      }
    })));
  };
}

function switchAll(e) {
  return switchMap(identity)(e);
}

function take(r) {
  return t => i => {
    var a = e;
    var f = !1;
    var n = 0;
    t((e => {
      if (f) {} else if (0 === e) {
        f = !0;
        i(0);
      } else if (0 === e.tag) {
        if (r <= 0) {
          f = !0;
          i(0);
          e[0](1);
        } else {
          a = e[0];
        }
      } else if (n++ < r) {
        i(e);
        if (!f && n >= r) {
          f = !0;
          i(0);
          a(1);
        }
      } else {
        i(e);
      }
    }));
    i(start((e => {
      if (1 === e && !f) {
        f = !0;
        a(1);
      } else if (0 === e && !f && n < r) {
        a(0);
      }
    })));
  };
}

function takeLast(t) {
  return i => a => {
    var f = [];
    var n = e;
    i((e => {
      if (0 === e) {
        r(f)(a);
      } else if (0 === e.tag) {
        if (t <= 0) {
          e[0](1);
          r(f)(a);
        } else {
          (n = e[0])(0);
        }
      } else {
        if (f.length >= t && t) {
          f.shift();
        }
        f.push(e[0]);
        n(0);
      }
    }));
  };
}

function takeUntil(r) {
  return t => i => {
    var a = e;
    var f = e;
    var n = !1;
    t((e => {
      if (n) {} else if (0 === e) {
        n = !0;
        f(1);
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        r((e => {
          if (0 === e) {} else if (0 === e.tag) {
            (f = e[0])(0);
          } else {
            n = !0;
            f(1);
            a(1);
            i(0);
          }
        }));
      } else {
        i(e);
      }
    }));
    i(start((e => {
      if (1 === e && !n) {
        n = !0;
        a(1);
        f(1);
      } else if (!n) {
        a(0);
      }
    })));
  };
}

function takeWhile(r, t) {
  return i => a => {
    var f = e;
    var n = !1;
    i((e => {
      if (n) {} else if (0 === e) {
        n = !0;
        a(0);
      } else if (0 === e.tag) {
        f = e[0];
        a(e);
      } else if (!r(e[0])) {
        n = !0;
        if (t) {
          a(e);
        }
        a(0);
        f(1);
      } else {
        a(e);
      }
    }));
  };
}

function debounce(e) {
  return r => t => {
    var i;
    var a = !1;
    var f = !1;
    r((r => {
      if (f) {} else if (0 === r) {
        f = !0;
        if (i) {
          a = !0;
        } else {
          t(0);
        }
      } else if (0 === r.tag) {
        var n = r[0];
        t(start((e => {
          if (1 === e && !f) {
            f = !0;
            a = !1;
            if (i) {
              clearTimeout(i);
            }
            n(1);
          } else if (!f) {
            n(0);
          }
        })));
      } else {
        if (i) {
          clearTimeout(i);
        }
        i = setTimeout((() => {
          i = void 0;
          t(r);
          if (a) {
            t(0);
          }
        }), e(r[0]));
      }
    }));
  };
}

function delay(e) {
  return r => t => {
    var i = 0;
    r((r => {
      if (0 !== r && 0 === r.tag) {
        t(r);
      } else {
        i++;
        setTimeout((() => {
          if (i) {
            i--;
            t(r);
          }
        }), e);
      }
    }));
  };
}

function throttle(e) {
  return r => t => {
    var i = !1;
    var a;
    r((r => {
      if (0 === r) {
        if (a) {
          clearTimeout(a);
        }
        t(0);
      } else if (0 === r.tag) {
        var f = r[0];
        t(start((e => {
          if (1 === e) {
            if (a) {
              clearTimeout(a);
            }
            f(1);
          } else {
            f(0);
          }
        })));
      } else if (!i) {
        i = !0;
        if (a) {
          clearTimeout(a);
        }
        a = setTimeout((() => {
          a = void 0;
          i = !1;
        }), e(r[0]));
        t(r);
      }
    }));
  };
}

function lazy(e) {
  return r => e()(r);
}

function fromAsyncIterable(e) {
  return r => {
    var t = e[asyncIteratorSymbol()] && e[asyncIteratorSymbol()]() || e;
    var i = !1;
    var a = !1;
    var f = !1;
    var n;
    r(start((async e => {
      if (1 === e) {
        i = !0;
        if (t.return) {
          t.return();
        }
      } else if (a) {
        f = !0;
      } else {
        for (f = a = !0; f && !i; ) {
          if ((n = await t.next()).done) {
            i = !0;
            if (t.return) {
              await t.return();
            }
            r(0);
          } else {
            try {
              f = !1;
              r(push(n.value));
            } catch (e) {
              if (t.throw) {
                if (i = !!(await t.throw(e)).done) {
                  r(0);
                }
              } else {
                throw e;
              }
            }
          }
        }
        a = !1;
      }
    })));
  };
}

function fromIterable(e) {
  if (e[Symbol.asyncIterator]) {
    return fromAsyncIterable(e);
  }
  return r => {
    var t = e[Symbol.iterator]();
    var i = !1;
    var a = !1;
    var f = !1;
    var n;
    r(start((e => {
      if (1 === e) {
        i = !0;
        if (t.return) {
          t.return();
        }
      } else if (a) {
        f = !0;
      } else {
        for (f = a = !0; f && !i; ) {
          if ((n = t.next()).done) {
            i = !0;
            if (t.return) {
              t.return();
            }
            r(0);
          } else {
            try {
              f = !1;
              r(push(n.value));
            } catch (e) {
              if (t.throw) {
                if (i = !!t.throw(e).done) {
                  r(0);
                }
              } else {
                throw e;
              }
            }
          }
        }
        a = !1;
      }
    })));
  };
}

var r = fromIterable;

function fromValue(e) {
  return r => {
    var t = !1;
    r(start((i => {
      if (1 === i) {
        t = !0;
      } else if (!t) {
        t = !0;
        r(push(e));
        r(0);
      }
    })));
  };
}

function make(e) {
  return r => {
    var t = !1;
    var i = e({
      next(e) {
        if (!t) {
          r(push(e));
        }
      },
      complete() {
        if (!t) {
          t = !0;
          r(0);
        }
      }
    });
    r(start((e => {
      if (1 === e && !t) {
        t = !0;
        i();
      }
    })));
  };
}

function makeSubject() {
  var e;
  var r;
  return {
    source: share(make((t => {
      e = t.next;
      r = t.complete;
      return teardownPlaceholder;
    }))),
    next(r) {
      if (e) {
        e(r);
      }
    },
    complete() {
      if (r) {
        r();
      }
    }
  };
}

var empty = e => {
  var r = !1;
  e(start((t => {
    if (1 === t) {
      r = !0;
    } else if (!r) {
      r = !0;
      e(0);
    }
  })));
};

var never = r => {
  r(start(e));
};

function interval(e) {
  return make((r => {
    var t = 0;
    var i = setInterval((() => r.next(t++)), e);
    return () => clearInterval(i);
  }));
}

function fromDomEvent(e, r) {
  return make((t => {
    e.addEventListener(r, t.next);
    return () => e.removeEventListener(r, t.next);
  }));
}

function fromPromise(e) {
  return make((r => {
    e.then((e => {
      Promise.resolve(e).then((() => {
        r.next(e);
        r.complete();
      }));
    }));
    return teardownPlaceholder;
  }));
}

function subscribe(r) {
  return t => {
    var i = e;
    var a = !1;
    t((e => {
      if (0 === e) {
        a = !0;
      } else if (0 === e.tag) {
        (i = e[0])(0);
      } else if (!a) {
        r(e[0]);
        i(0);
      }
    }));
    return {
      unsubscribe() {
        if (!a) {
          a = !0;
          i(1);
        }
      }
    };
  };
}

function forEach(e) {
  return r => {
    subscribe(e)(r);
  };
}

function publish(e) {
  subscribe((e => {}))(e);
}

var t = {
  done: !0
};

var toAsyncIterable = r => {
  var i = [];
  var a = !1;
  var f = !1;
  var n = !1;
  var s = e;
  var l;
  return {
    async next() {
      if (!f) {
        f = !0;
        r((e => {
          if (a) {} else if (0 === e) {
            if (l) {
              l = l(t);
            }
            a = !0;
          } else if (0 === e.tag) {
            n = !0;
            (s = e[0])(0);
          } else {
            n = !1;
            if (l) {
              l = l({
                value: e[0],
                done: !1
              });
            } else {
              i.push(e[0]);
            }
          }
        }));
      }
      if (a && !i.length) {
        return t;
      } else if (!a && !n && i.length <= 1) {
        n = !0;
        s(0);
      }
      return i.length ? {
        value: i.shift(),
        done: !1
      } : new Promise((e => l = e));
    },
    async return() {
      if (!a) {
        l = s(1);
      }
      a = !0;
      return t;
    },
    [asyncIteratorSymbol()]() {
      return this;
    }
  };
};

function toArray(r) {
  var t = [];
  var i = e;
  var a = !1;
  r((e => {
    if (0 === e) {
      a = !0;
    } else if (0 === e.tag) {
      (i = e[0])(0);
    } else {
      t.push(e[0]);
      i(0);
    }
  }));
  if (!a) {
    i(1);
  }
  return t;
}

function toPromise(r) {
  return new Promise((t => {
    var i = e;
    var a;
    r((e => {
      if (0 === e) {
        Promise.resolve(a).then(t);
      } else if (0 === e.tag) {
        (i = e[0])(0);
      } else {
        a = e[0];
        i(0);
      }
    }));
  }));
}

function zip(r) {
  var t = Object.keys(r).length;
  return i => {
    var a = new Set;
    var f = Array.isArray(r) ? new Array(t).fill(e) : {};
    var n = Array.isArray(r) ? new Array(t) : {};
    var s = !1;
    var l = !1;
    var u = !1;
    var o = 0;
    var loop = function(v) {
      r[v]((c => {
        if (0 === c) {
          if (o >= t - 1) {
            u = !0;
            i(0);
          } else {
            o++;
          }
        } else if (0 === c.tag) {
          f[v] = c[0];
        } else if (!u) {
          n[v] = c[0];
          a.add(v);
          if (!s && a.size < t) {
            if (!l) {
              for (var h in r) {
                if (!a.has(h)) {
                  (f[h] || e)(0);
                }
              }
            } else {
              l = !1;
            }
          } else {
            s = !0;
            l = !1;
            i(push(Array.isArray(n) ? n.slice() : {
              ...n
            }));
          }
        }
      }));
    };
    for (var v in r) {
      loop(v);
    }
    i(start((e => {
      if (u) {} else if (1 === e) {
        u = !0;
        for (var r in f) {
          f[r](1);
        }
      } else if (!l) {
        l = !0;
        for (var t in f) {
          f[t](0);
        }
      }
    })));
  };
}

function combine(...e) {
  return zip(e);
}

function fromObservable(e) {
  return r => {
    var t = (e[observableSymbol()] ? e[observableSymbol()]() : e).subscribe({
      next(e) {
        r(push(e));
      },
      complete() {
        r(0);
      },
      error(e) {
        throw e;
      }
    });
    r(start((e => {
      if (1 === e) {
        t.unsubscribe();
      }
    })));
  };
}

function toObservable(r) {
  return {
    subscribe(t, i, a) {
      var f = "object" == typeof t ? t : {
        next: t,
        error: i,
        complete: a
      };
      var n = e;
      var s = !1;
      r((e => {
        if (s) {} else if (0 === e) {
          s = !0;
          if (f.complete) {
            f.complete();
          }
        } else if (0 === e.tag) {
          (n = e[0])(0);
        } else {
          f.next(e[0]);
          n(0);
        }
      }));
      var l = {
        closed: !1,
        unsubscribe() {
          l.closed = !0;
          s = !0;
          n(1);
        }
      };
      return l;
    },
    [observableSymbol()]() {
      return this;
    }
  };
}

function fromCallbag(e) {
  return r => {
    e(0, ((e, t) => {
      if (0 === e) {
        r(start((e => {
          t(e + 1);
        })));
      } else if (1 === e) {
        r(push(t));
      } else {
        r(0);
      }
    }));
  };
}

function toCallbag(e) {
  return (r, t) => {
    if (0 === r) {
      e((e => {
        if (0 === e) {
          t(2);
        } else if (0 === e.tag) {
          t(0, (r => {
            if (r < 3) {
              e[0](r - 1);
            }
          }));
        } else {
          t(1, e[0]);
        }
      }));
    }
  };
}

var pipe = (...e) => {
  var r = e[0];
  for (var t = 1, i = e.length; t < i; t++) {
    r = e[t](r);
  }
  return r;
};

export { buffer, combine, concat, concatAll, concatMap, debounce, delay, empty, filter, mergeAll as flatten, forEach, r as fromArray, fromAsyncIterable, fromCallbag, fromDomEvent, fromIterable, fromObservable, fromPromise, fromValue, interval, lazy, make, makeSubject, map, merge, mergeAll, mergeMap, never, onEnd, onPush, onStart, pipe, publish, sample, scan, share, skip, skipUntil, skipWhile, subscribe, switchAll, switchMap, take, takeLast, takeUntil, takeWhile, onPush as tap, throttle, toArray, toAsyncIterable, toCallbag, toObservable, toPromise, zip };
//# sourceMappingURL=wonka.mjs.map
