'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var utils = require('@chakra-ui/utils');
var React = require('react');

function _interopNamespace(e) {
  if (e && e.__esModule) return e;
  var n = Object.create(null);
  if (e) {
    Object.keys(e).forEach(function (k) {
      if (k !== 'default') {
        var d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: function () { return e[k]; }
        });
      }
    });
  }
  n["default"] = e;
  return Object.freeze(n);
}

var React__namespace = /*#__PURE__*/_interopNamespace(React);

var LiveRegion = /*#__PURE__*/function () {
  function LiveRegion(options) {
    this.region = void 0;
    this.options = void 0;
    this.parentNode = void 0;
    this.options = getOptions(options);
    this.region = getRegion(this.options);
    this.parentNode = this.options.parentNode;

    if (this.region) {
      this.parentNode.appendChild(this.region);
    }
  }

  var _proto = LiveRegion.prototype;

  _proto.speak = function speak(message) {
    this.clear();

    if (this.region) {
      this.region.innerText = message;
    }
  };

  _proto.destroy = function destroy() {
    if (this.region) {
      var _this$region$parentNo;

      (_this$region$parentNo = this.region.parentNode) == null ? void 0 : _this$region$parentNo.removeChild(this.region);
    }
  };

  _proto.clear = function clear() {
    if (this.region) {
      this.region.innerText = "";
    }
  };

  return LiveRegion;
}();

function getOptions(options) {
  var defaultOptions = {
    "aria-live": "polite",
    "aria-atomic": "true",
    "aria-relevant": "all",
    role: "status",
    id: "chakra-a11y-live-region",
    parentNode: utils.isBrowser ? document.body : undefined
  };

  if (options) {
    return Object.assign(defaultOptions, options);
  }

  return defaultOptions;
}

function getRegion(options) {
  var region = utils.isBrowser ? document.getElementById(options.id) : null;
  if (region) return region;

  if (utils.isBrowser) {
    region = document.createElement("div");
    setup(region, options);
  }

  return region;
}

function setup(region, options) {
  region.id = options.id || "chakra-live-region";
  region.className = "__chakra-live-region";
  region.setAttribute("aria-live", options["aria-live"]);
  region.setAttribute("role", options.role);
  region.setAttribute("aria-relevant", options["aria-relevant"]);
  region.setAttribute("aria-atomic", String(options["aria-atomic"]));
  Object.assign(region.style, {
    border: "0px",
    clip: "rect(0px, 0px, 0px, 0px)",
    height: "1px",
    width: "1px",
    margin: "-1px",
    padding: "0px",
    overflow: "hidden",
    whiteSpace: "nowrap",
    position: "absolute"
  });
}

function useLiveRegion(options) {
  var _React$useState = React__namespace.useState(function () {
    return new LiveRegion(options);
  }),
      liveRegion = _React$useState[0];

  React__namespace.useEffect(function () {
    return function () {
      liveRegion.destroy();
    };
  }, [liveRegion]);
  return liveRegion;
}

exports.LiveRegion = LiveRegion;
exports.useLiveRegion = useLiveRegion;
