import { __extends, __rest, __read, __assign } from 'tslib';
import * as React from 'react';
import { forwardRef, useContext, useRef, useLayoutEffect } from 'react';
import { MotionContext } from '../../context/MotionContext/index.mjs';
import mergeRefs from 'react-merge-refs';
import { render, events, unmountComponentAtNode } from '@react-three/fiber';
import { useIsomorphicLayoutEffect } from '../../utils/use-isomorphic-effect.mjs';
import { MotionConfigContext } from '../../context/MotionConfigContext.mjs';
import { MotionCanvasContext } from './MotionCanvasContext.mjs';
import { useForceUpdate } from '../../utils/use-force-update.mjs';
import { clamp } from 'popmotion';

var devicePixelRatio = typeof window !== "undefined" ? window.devicePixelRatio : 1;
var calculateDpr = function (dpr) {
    return Array.isArray(dpr)
        ? clamp(dpr[0], dpr[1], devicePixelRatio)
        : dpr || devicePixelRatio;
};
/**
 * This file contains a version of R3F's Canvas component that uses our projection
 * system for layout measurements instead of use-react-measure so we can keep the
 * projection and cameras in sync.
 *
 * https://github.com/pmndrs/react-three-fiber/blob/master/packages/fiber/src/web/Canvas.tsx
 */
function Block(_a) {
    var set = _a.set;
    useIsomorphicLayoutEffect(function () {
        set(new Promise(function () { return null; }));
        return function () { return set(false); };
    }, []);
    return null;
}
var ErrorBoundary = /** @class */ (function (_super) {
    __extends(ErrorBoundary, _super);
    function ErrorBoundary() {
        var _this = _super !== null && _super.apply(this, arguments) || this;
        _this.state = { error: false };
        return _this;
    }
    ErrorBoundary.prototype.componentDidCatch = function (error) {
        this.props.set(error);
    };
    ErrorBoundary.prototype.render = function () {
        return this.state.error ? null : this.props.children;
    };
    ErrorBoundary.getDerivedStateFromError = function () { return ({ error: true }); };
    return ErrorBoundary;
}(React.Component));
function CanvasComponent(_a, forwardedRef) {
    var children = _a.children, fallback = _a.fallback, tabIndex = _a.tabIndex, id = _a.id, style = _a.style, className = _a.className, events$1 = _a.events, props = __rest(_a, ["children", "fallback", "tabIndex", "id", "style", "className", "events"]);
    /**
     * Import existing contexts to pass through variants and MotionConfig from
     * the DOM to the 3D tree. Shared variants aren't officially supported yet
     * because the parent DOM tree fires effects before the 3D tree, whereas
     * variants are expected to run from bottom-up in useEffect.
     */
    var motionContext = useContext(MotionContext);
    var configContext = useContext(MotionConfigContext);
    var _b = __read(useForceUpdate(), 1), forceRender = _b[0];
    var layoutCamera = useRef(null);
    var dimensions = useRef({
        size: { width: 0, height: 0 },
    });
    var _c = dimensions.current, size = _c.size, dpr = _c.dpr;
    var containerRef = useRef(null);
    var handleResize = function () {
        var container = containerRef.current;
        dimensions.current = {
            size: {
                width: container.offsetWidth,
                height: container.offsetHeight,
            },
        };
        forceRender();
    };
    // Set canvas size on mount
    useLayoutEffect(handleResize, []);
    var canvasRef = React.useRef(null);
    var _d = __read(React.useState(false), 2), block = _d[0], setBlock = _d[1];
    var _e = __read(React.useState(false), 2), error = _e[0], setError = _e[1];
    // Suspend this component if block is a promise (2nd run)
    if (block)
        throw block;
    // Throw exception outwards if anything within canvas throws
    if (error)
        throw error;
    // Only render the R3F tree once we have recorded dimensions for the canvas.
    if (size.width > 0 && size.height > 0) {
        render(React.createElement(ErrorBoundary, { set: setError },
            React.createElement(React.Suspense, { fallback: React.createElement(Block, { set: setBlock }) },
                React.createElement(MotionCanvasContext.Provider, { value: {
                        dimensions: dimensions,
                        layoutCamera: layoutCamera,
                        requestedDpr: calculateDpr(props.dpr),
                    } },
                    React.createElement(MotionConfigContext.Provider, { value: configContext },
                        React.createElement(MotionContext.Provider, { value: motionContext }, children))))), canvasRef.current, __assign(__assign({}, props), { dpr: dpr || props.dpr, size: size, events: events$1 || events }));
    }
    useIsomorphicLayoutEffect(function () {
        var container = canvasRef.current;
        return function () { return unmountComponentAtNode(container); };
    }, []);
    return (React.createElement("div", { ref: containerRef, id: id, className: className, tabIndex: tabIndex, style: __assign({ position: "relative", width: "100%", height: "100%", overflow: "hidden" }, style) },
        React.createElement("canvas", { ref: mergeRefs([canvasRef, forwardedRef]), style: { display: "block" } }, fallback)));
}
var MotionCanvas = forwardRef(CanvasComponent);

export { MotionCanvas };
