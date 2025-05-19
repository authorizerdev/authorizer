Object.defineProperty(exports, "__esModule", {
  value: !0
});

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

function concatMap(r) {
  return t => i => {
    var s = [];
    var a = e;
    var f = e;
    var n = !1;
    var l = !1;
    var o = !1;
    var u = !1;
    function applyInnerSource(e) {
      o = !0;
      e((e => {
        if (0 === e) {
          if (o) {
            o = !1;
            if (s.length) {
              applyInnerSource(r(s.shift()));
            } else if (u) {
              i(0);
            } else if (!n) {
              n = !0;
              a(0);
            }
          }
        } else if (0 === e.tag) {
          l = !1;
          (f = e[0])(0);
        } else if (o) {
          i(e);
          if (l) {
            l = !1;
          } else {
            f(0);
          }
        }
      }));
    }
    t((e => {
      if (u) {} else if (0 === e) {
        u = !0;
        if (!o && !s.length) {
          i(0);
        }
      } else if (0 === e.tag) {
        a = e[0];
      } else {
        n = !1;
        if (o) {
          s.push(e[0]);
        } else {
          applyInnerSource(r(e[0]));
        }
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!u) {
          u = !0;
          a(1);
        }
        if (o) {
          o = !1;
          f(1);
        }
      } else {
        if (!u && !n) {
          n = !0;
          a(0);
        }
        if (o && !l) {
          l = !0;
          f(0);
        }
      }
    })));
  };
}

function concatAll(e) {
  return concatMap(identity)(e);
}

function mergeMap(r) {
  return t => i => {
    var s = [];
    var a = e;
    var f = !1;
    var n = !1;
    t((t => {
      if (n) {} else if (0 === t) {
        n = !0;
        if (!s.length) {
          i(0);
        }
      } else if (0 === t.tag) {
        a = t[0];
      } else {
        f = !1;
        !function applyInnerSource(r) {
          var t = e;
          r((e => {
            if (0 === e) {
              if (s.length) {
                var r = s.indexOf(t);
                if (r > -1) {
                  (s = s.slice()).splice(r, 1);
                }
                if (!s.length) {
                  if (n) {
                    i(0);
                  } else if (!f) {
                    f = !0;
                    a(0);
                  }
                }
              }
            } else if (0 === e.tag) {
              s.push(t = e[0]);
              t(0);
            } else if (s.length) {
              i(e);
              t(0);
            }
          }));
        }(r(t[0]));
        if (!f) {
          f = !0;
          a(0);
        }
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!n) {
          n = !0;
          a(1);
        }
        for (var r = 0, t = s, i = s.length; r < i; r++) {
          t[r](1);
        }
        s.length = 0;
      } else {
        if (!n && !f) {
          f = !0;
          a(0);
        } else {
          f = !1;
        }
        for (var l = 0, o = s, u = s.length; l < u; l++) {
          o[l](0);
        }
      }
    })));
  };
}

function mergeAll(e) {
  return mergeMap(identity)(e);
}

function onPush(e) {
  return r => t => {
    var i = !1;
    r((r => {
      if (i) {} else if (0 === r) {
        i = !0;
        t(0);
      } else if (0 === r.tag) {
        var s = r[0];
        t(start((e => {
          if (1 === e) {
            i = !0;
          }
          s(e);
        })));
      } else {
        e(r[0]);
        t(r);
      }
    }));
  };
}

function share(r) {
  var t = [];
  var i = e;
  var s = !1;
  return e => {
    t.push(e);
    if (1 === t.length) {
      r((e => {
        if (0 === e) {
          for (var r = 0, a = t, f = t.length; r < f; r++) {
            a[r](0);
          }
          t.length = 0;
        } else if (0 === e.tag) {
          i = e[0];
        } else {
          s = !1;
          for (var n = 0, l = t, o = t.length; n < o; n++) {
            l[n](e);
          }
        }
      }));
    }
    e(start((r => {
      if (1 === r) {
        var a = t.indexOf(e);
        if (a > -1) {
          (t = t.slice()).splice(a, 1);
        }
        if (!t.length) {
          i(1);
        }
      } else if (!s) {
        s = !0;
        i(0);
      }
    })));
  };
}

function switchMap(r) {
  return t => i => {
    var s = e;
    var a = e;
    var f = !1;
    var n = !1;
    var l = !1;
    var o = !1;
    t((t => {
      if (o) {} else if (0 === t) {
        o = !0;
        if (!l) {
          i(0);
        }
      } else if (0 === t.tag) {
        s = t[0];
      } else {
        if (l) {
          a(1);
          a = e;
        }
        if (!f) {
          f = !0;
          s(0);
        } else {
          f = !1;
        }
        !function applyInnerSource(e) {
          l = !0;
          e((e => {
            if (!l) {} else if (0 === e) {
              l = !1;
              if (o) {
                i(0);
              } else if (!f) {
                f = !0;
                s(0);
              }
            } else if (0 === e.tag) {
              n = !1;
              (a = e[0])(0);
            } else {
              i(e);
              if (!n) {
                a(0);
              } else {
                n = !1;
              }
            }
          }));
        }(r(t[0]));
      }
    }));
    i(start((e => {
      if (1 === e) {
        if (!o) {
          o = !0;
          s(1);
        }
        if (l) {
          l = !1;
          a(1);
        }
      } else {
        if (!o && !f) {
          f = !0;
          s(0);
        }
        if (l && !n) {
          n = !0;
          a(0);
        }
      }
    })));
  };
}

function fromAsyncIterable(e) {
  return r => {
    var t = e[asyncIteratorSymbol()] && e[asyncIteratorSymbol()]() || e;
    var i = !1;
    var s = !1;
    var a = !1;
    var f;
    r(start((async e => {
      if (1 === e) {
        i = !0;
        if (t.return) {
          t.return();
        }
      } else if (s) {
        a = !0;
      } else {
        for (a = s = !0; a && !i; ) {
          if ((f = await t.next()).done) {
            i = !0;
            if (t.return) {
              await t.return();
            }
            r(0);
          } else {
            try {
              a = !1;
              r(push(f.value));
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
        s = !1;
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
    var s = !1;
    var a = !1;
    var f;
    r(start((e => {
      if (1 === e) {
        i = !0;
        if (t.return) {
          t.return();
        }
      } else if (s) {
        a = !0;
      } else {
        for (a = s = !0; a && !i; ) {
          if ((f = t.next()).done) {
            i = !0;
            if (t.return) {
              t.return();
            }
            r(0);
          } else {
            try {
              a = !1;
              r(push(f.value));
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
        s = !1;
      }
    })));
  };
}

var r = fromIterable;

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

function subscribe(r) {
  return t => {
    var i = e;
    var s = !1;
    t((e => {
      if (0 === e) {
        s = !0;
      } else if (0 === e.tag) {
        (i = e[0])(0);
      } else if (!s) {
        r(e[0]);
        i(0);
      }
    }));
    return {
      unsubscribe() {
        if (!s) {
          s = !0;
          i(1);
        }
      }
    };
  };
}

var t = {
  done: !0
};

function zip(r) {
  var t = Object.keys(r).length;
  return i => {
    var s = new Set;
    var a = Array.isArray(r) ? new Array(t).fill(e) : {};
    var f = Array.isArray(r) ? new Array(t) : {};
    var n = !1;
    var l = !1;
    var o = !1;
    var u = 0;
    var loop = function(v) {
      r[v]((c => {
        if (0 === c) {
          if (u >= t - 1) {
            o = !0;
            i(0);
          } else {
            u++;
          }
        } else if (0 === c.tag) {
          a[v] = c[0];
        } else if (!o) {
          f[v] = c[0];
          s.add(v);
          if (!n && s.size < t) {
            if (!l) {
              for (var p in r) {
                if (!s.has(p)) {
                  (a[p] || e)(0);
                }
              }
            } else {
              l = !1;
            }
          } else {
            n = !0;
            l = !1;
            i(push(Array.isArray(f) ? f.slice() : {
              ...f
            }));
          }
        }
      }));
    };
    for (var v in r) {
      loop(v);
    }
    i(start((e => {
      if (o) {} else if (1 === e) {
        o = !0;
        for (var r in a) {
          a[r](1);
        }
      } else if (!l) {
        l = !0;
        for (var t in a) {
          a[t](0);
        }
      }
    })));
  };
}

exports.buffer = function buffer(r) {
  return t => i => {
    var s = [];
    var a = e;
    var f = e;
    var n = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        f(1);
        if (s.length) {
          i(push(s));
        }
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
        r((e => {
          if (l) {} else if (0 === e) {
            l = !0;
            a(1);
            if (s.length) {
              i(push(s));
            }
            i(0);
          } else if (0 === e.tag) {
            f = e[0];
          } else if (s.length) {
            var r = push(s);
            s = [];
            i(r);
          }
        }));
      } else {
        s.push(e[0]);
        if (!n) {
          n = !0;
          a(0);
          f(0);
        } else {
          n = !1;
        }
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        a(1);
        f(1);
      } else if (!l && !n) {
        n = !0;
        a(0);
        f(0);
      }
    })));
  };
};

exports.combine = function combine(...e) {
  return zip(e);
};

exports.concat = function concat(e) {
  return concatAll(r(e));
};

exports.concatAll = concatAll;

exports.concatMap = concatMap;

exports.debounce = function debounce(e) {
  return r => t => {
    var i;
    var s = !1;
    var a = !1;
    r((r => {
      if (a) {} else if (0 === r) {
        a = !0;
        if (i) {
          s = !0;
        } else {
          t(0);
        }
      } else if (0 === r.tag) {
        var f = r[0];
        t(start((e => {
          if (1 === e && !a) {
            a = !0;
            s = !1;
            if (i) {
              clearTimeout(i);
            }
            f(1);
          } else if (!a) {
            f(0);
          }
        })));
      } else {
        if (i) {
          clearTimeout(i);
        }
        i = setTimeout((() => {
          i = void 0;
          t(r);
          if (s) {
            t(0);
          }
        }), e(r[0]));
      }
    }));
  };
};

exports.delay = function delay(e) {
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
};

exports.empty = e => {
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

exports.filter = function filter(r) {
  return t => i => {
    var s = e;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
        i(e);
      } else if (!r(e[0])) {
        s(0);
      } else {
        i(e);
      }
    }));
  };
};

exports.flatten = mergeAll;

exports.forEach = function forEach(e) {
  return r => {
    subscribe(e)(r);
  };
};

exports.fromArray = r;

exports.fromAsyncIterable = fromAsyncIterable;

exports.fromCallbag = function fromCallbag(e) {
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
};

exports.fromDomEvent = function fromDomEvent(e, r) {
  return make((t => {
    e.addEventListener(r, t.next);
    return () => e.removeEventListener(r, t.next);
  }));
};

exports.fromIterable = fromIterable;

exports.fromObservable = function fromObservable(e) {
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
};

exports.fromPromise = function fromPromise(e) {
  return make((r => {
    e.then((e => {
      Promise.resolve(e).then((() => {
        r.next(e);
        r.complete();
      }));
    }));
    return teardownPlaceholder;
  }));
};

exports.fromValue = function fromValue(e) {
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
};

exports.interval = function interval(e) {
  return make((r => {
    var t = 0;
    var i = setInterval((() => r.next(t++)), e);
    return () => clearInterval(i);
  }));
};

exports.lazy = function lazy(e) {
  return r => e()(r);
};

exports.make = make;

exports.makeSubject = function makeSubject() {
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
};

exports.map = function map(e) {
  return r => t => r((r => {
    if (0 === r || 0 === r.tag) {
      t(r);
    } else {
      t(push(e(r[0])));
    }
  }));
};

exports.merge = function merge(e) {
  return mergeAll(r(e));
};

exports.mergeAll = mergeAll;

exports.mergeMap = mergeMap;

exports.never = r => {
  r(start(e));
};

exports.onEnd = function onEnd(e) {
  return r => t => {
    var i = !1;
    r((r => {
      if (i) {} else if (0 === r) {
        i = !0;
        t(0);
        e();
      } else if (0 === r.tag) {
        var s = r[0];
        t(start((r => {
          if (1 === r) {
            i = !0;
            s(1);
            e();
          } else {
            s(r);
          }
        })));
      } else {
        t(r);
      }
    }));
  };
};

exports.onPush = onPush;

exports.onStart = function onStart(e) {
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
};

exports.pipe = (...e) => {
  var r = e[0];
  for (var t = 1, i = e.length; t < i; t++) {
    r = e[t](r);
  }
  return r;
};

exports.publish = function publish(e) {
  subscribe((e => {}))(e);
};

exports.sample = function sample(r) {
  return t => i => {
    var s = e;
    var a = e;
    var f;
    var n = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        a(1);
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
      } else {
        f = e[0];
        if (!n) {
          n = !0;
          a(0);
          s(0);
        } else {
          n = !1;
        }
      }
    }));
    r((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        s(1);
        i(0);
      } else if (0 === e.tag) {
        a = e[0];
      } else if (void 0 !== f) {
        var r = push(f);
        f = void 0;
        i(r);
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        s(1);
        a(1);
      } else if (!l && !n) {
        n = !0;
        s(0);
        a(0);
      }
    })));
  };
};

exports.scan = function scan(e, r) {
  return t => i => {
    var s = r;
    t((r => {
      if (0 === r) {
        i(0);
      } else if (0 === r.tag) {
        i(r);
      } else {
        i(push(s = e(s, r[0])));
      }
    }));
  };
};

exports.share = share;

exports.skip = function skip(r) {
  return t => i => {
    var s = e;
    var a = r;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
        i(e);
      } else if (a-- > 0) {
        s(0);
      } else {
        i(e);
      }
    }));
  };
};

exports.skipUntil = function skipUntil(r) {
  return t => i => {
    var s = e;
    var a = e;
    var f = !0;
    var n = !1;
    var l = !1;
    t((e => {
      if (l) {} else if (0 === e) {
        l = !0;
        if (f) {
          a(1);
        }
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
        r((e => {
          if (0 === e) {
            if (f) {
              l = !0;
              s(1);
            }
          } else if (0 === e.tag) {
            (a = e[0])(0);
          } else {
            f = !1;
            a(1);
          }
        }));
      } else if (!f) {
        n = !1;
        i(e);
      } else if (!n) {
        n = !0;
        s(0);
        a(0);
      } else {
        n = !1;
      }
    }));
    i(start((e => {
      if (1 === e && !l) {
        l = !0;
        s(1);
        if (f) {
          a(1);
        }
      } else if (!l && !n) {
        n = !0;
        if (f) {
          a(0);
        }
        s(0);
      }
    })));
  };
};

exports.skipWhile = function skipWhile(r) {
  return t => i => {
    var s = e;
    var a = !0;
    t((e => {
      if (0 === e) {
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
        i(e);
      } else if (a) {
        if (r(e[0])) {
          s(0);
        } else {
          a = !1;
          i(e);
        }
      } else {
        i(e);
      }
    }));
  };
};

exports.subscribe = subscribe;

exports.switchAll = function switchAll(e) {
  return switchMap(identity)(e);
};

exports.switchMap = switchMap;

exports.take = function take(r) {
  return t => i => {
    var s = e;
    var a = !1;
    var f = 0;
    t((e => {
      if (a) {} else if (0 === e) {
        a = !0;
        i(0);
      } else if (0 === e.tag) {
        if (r <= 0) {
          a = !0;
          i(0);
          e[0](1);
        } else {
          s = e[0];
        }
      } else if (f++ < r) {
        i(e);
        if (!a && f >= r) {
          a = !0;
          i(0);
          s(1);
        }
      } else {
        i(e);
      }
    }));
    i(start((e => {
      if (1 === e && !a) {
        a = !0;
        s(1);
      } else if (0 === e && !a && f < r) {
        s(0);
      }
    })));
  };
};

exports.takeLast = function takeLast(t) {
  return i => s => {
    var a = [];
    var f = e;
    i((e => {
      if (0 === e) {
        r(a)(s);
      } else if (0 === e.tag) {
        if (t <= 0) {
          e[0](1);
          r(a)(s);
        } else {
          (f = e[0])(0);
        }
      } else {
        if (a.length >= t && t) {
          a.shift();
        }
        a.push(e[0]);
        f(0);
      }
    }));
  };
};

exports.takeUntil = function takeUntil(r) {
  return t => i => {
    var s = e;
    var a = e;
    var f = !1;
    t((e => {
      if (f) {} else if (0 === e) {
        f = !0;
        a(1);
        i(0);
      } else if (0 === e.tag) {
        s = e[0];
        r((e => {
          if (0 === e) {} else if (0 === e.tag) {
            (a = e[0])(0);
          } else {
            f = !0;
            a(1);
            s(1);
            i(0);
          }
        }));
      } else {
        i(e);
      }
    }));
    i(start((e => {
      if (1 === e && !f) {
        f = !0;
        s(1);
        a(1);
      } else if (!f) {
        s(0);
      }
    })));
  };
};

exports.takeWhile = function takeWhile(r, t) {
  return i => s => {
    var a = e;
    var f = !1;
    i((e => {
      if (f) {} else if (0 === e) {
        f = !0;
        s(0);
      } else if (0 === e.tag) {
        a = e[0];
        s(e);
      } else if (!r(e[0])) {
        f = !0;
        if (t) {
          s(e);
        }
        s(0);
        a(1);
      } else {
        s(e);
      }
    }));
  };
};

exports.tap = onPush;

exports.throttle = function throttle(e) {
  return r => t => {
    var i = !1;
    var s;
    r((r => {
      if (0 === r) {
        if (s) {
          clearTimeout(s);
        }
        t(0);
      } else if (0 === r.tag) {
        var a = r[0];
        t(start((e => {
          if (1 === e) {
            if (s) {
              clearTimeout(s);
            }
            a(1);
          } else {
            a(0);
          }
        })));
      } else if (!i) {
        i = !0;
        if (s) {
          clearTimeout(s);
        }
        s = setTimeout((() => {
          s = void 0;
          i = !1;
        }), e(r[0]));
        t(r);
      }
    }));
  };
};

exports.toArray = function toArray(r) {
  var t = [];
  var i = e;
  var s = !1;
  r((e => {
    if (0 === e) {
      s = !0;
    } else if (0 === e.tag) {
      (i = e[0])(0);
    } else {
      t.push(e[0]);
      i(0);
    }
  }));
  if (!s) {
    i(1);
  }
  return t;
};

exports.toAsyncIterable = r => {
  var i = [];
  var s = !1;
  var a = !1;
  var f = !1;
  var n = e;
  var l;
  return {
    async next() {
      if (!a) {
        a = !0;
        r((e => {
          if (s) {} else if (0 === e) {
            if (l) {
              l = l(t);
            }
            s = !0;
          } else if (0 === e.tag) {
            f = !0;
            (n = e[0])(0);
          } else {
            f = !1;
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
      if (s && !i.length) {
        return t;
      } else if (!s && !f && i.length <= 1) {
        f = !0;
        n(0);
      }
      return i.length ? {
        value: i.shift(),
        done: !1
      } : new Promise((e => l = e));
    },
    async return() {
      if (!s) {
        l = n(1);
      }
      s = !0;
      return t;
    },
    [asyncIteratorSymbol()]() {
      return this;
    }
  };
};

exports.toCallbag = function toCallbag(e) {
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
};

exports.toObservable = function toObservable(r) {
  return {
    subscribe(t, i, s) {
      var a = "object" == typeof t ? t : {
        next: t,
        error: i,
        complete: s
      };
      var f = e;
      var n = !1;
      r((e => {
        if (n) {} else if (0 === e) {
          n = !0;
          if (a.complete) {
            a.complete();
          }
        } else if (0 === e.tag) {
          (f = e[0])(0);
        } else {
          a.next(e[0]);
          f(0);
        }
      }));
      var l = {
        closed: !1,
        unsubscribe() {
          l.closed = !0;
          n = !0;
          f(1);
        }
      };
      return l;
    },
    [observableSymbol()]() {
      return this;
    }
  };
};

exports.toPromise = function toPromise(r) {
  return new Promise((t => {
    var i = e;
    var s;
    r((e => {
      if (0 === e) {
        Promise.resolve(s).then(t);
      } else if (0 === e.tag) {
        (i = e[0])(0);
      } else {
        s = e[0];
        i(0);
      }
    }));
  }));
};

exports.zip = zip;
//# sourceMappingURL=wonka.js.map
