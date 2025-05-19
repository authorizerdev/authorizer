"use strict";

var _typeof = require("@babel/runtime/helpers/typeof");
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports["default"] = void 0;
var _react = _interopRequireWildcard(require("react"));
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function _getRequireWildcardCache(e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != _typeof(e) && "function" != typeof e) return { "default": e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && Object.prototype.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n["default"] = e, t && t.set(e, n), n; }
function withSideEffect(reducePropsToState, handleStateChangeOnClient) {
  if (process.env.NODE_ENV !== 'production') {
    if (typeof reducePropsToState !== 'function') {
      throw new Error('Expected reducePropsToState to be a function.');
    }
    if (typeof handleStateChangeOnClient !== 'function') {
      throw new Error('Expected handleStateChangeOnClient to be a function.');
    }
  }
  return function wrap(WrappedComponent) {
    if (process.env.NODE_ENV !== 'production') {
      if (typeof WrappedComponent !== 'function') {
        throw new Error('Expected WrappedComponent to be a React component.');
      }
    }
    var mountedInstances = [];
    function emitChange() {
      console.log('emitting');
      var state = reducePropsToState(mountedInstances.map(function (instance) {
        return instance.current;
      }));
      handleStateChangeOnClient(state);
    }
    var SideEffect = function SideEffect(props) {
      var lastProps = (0, _react.useRef)(props);
      (0, _react.useEffect)(function () {
        lastProps.current = props;
      });
      (0, _react.useEffect)(function () {
        console.log('ins added');
        mountedInstances.push(lastProps);
        return function () {
          console.log('ins removed');
          var index = mountedInstances.indexOf(lastProps);
          mountedInstances.splice(index, 1);
        };
      }, []);
      return /*#__PURE__*/_react["default"].createElement(WrappedComponent, props);
    };
    return SideEffect;
  };
}
var _default = exports["default"] = withSideEffect;