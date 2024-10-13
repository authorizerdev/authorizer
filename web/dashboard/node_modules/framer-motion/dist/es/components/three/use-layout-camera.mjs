import { useContext, useRef, useLayoutEffect } from 'react';
import { useThree } from '@react-three/fiber';
import { useVisualElementContext } from '../../context/MotionContext/index.mjs';
import { MotionCanvasContext } from './MotionCanvasContext.mjs';
import { invariant } from 'hey-listen';
import { calcLength } from '../../projection/geometry/delta-calc.mjs';
import { clamp } from 'popmotion';

var calcBoxSize = function (_a) {
    var x = _a.x, y = _a.y;
    return ({
        width: calcLength(x),
        height: calcLength(y),
    });
};
function useLayoutCamera(_a, updateCamera) {
    var _b = _a.makeDefault, makeDefault = _b === void 0 ? true : _b;
    var context = useContext(MotionCanvasContext);
    invariant(Boolean(context), "No MotionCanvas detected. Replace Canvas from @react-three/fiber with MotionCanvas from framer-motion.");
    var _c = context, dimensions = _c.dimensions, layoutCamera = _c.layoutCamera, requestedDpr = _c.requestedDpr;
    var advance = useThree(function (three) { return three.advance; });
    var set = useThree(function (three) { return three.set; });
    var camera = useThree(function (three) { return three.camera; });
    var size = useThree(function (three) { return three.size; });
    var gl = useThree(function (three) { return three.gl; });
    var parentVisualElement = useVisualElementContext();
    var measuredLayoutSize = useRef();
    useLayoutEffect(function () {
        measuredLayoutSize.current = size;
        updateCamera(size);
        advance(performance.now());
        var projection = parentVisualElement === null || parentVisualElement === void 0 ? void 0 : parentVisualElement.projection;
        if (!projection)
            return;
        /**
         * When the projection of an element changes we want to update the camera
         * with the projected dimensions.
         */
        var removeProjectionUpdateListener = projection.addEventListener("projectionUpdate", function (newProjection) { return updateCamera(calcBoxSize(newProjection)); });
        /**
         * When the layout of an element changes we want to update the renderer
         * output to match the layout dimensions.
         */
        var removeLayoutMeasureListener = projection.addEventListener("measure", function (newLayout) {
            var newSize = calcBoxSize(newLayout);
            var dpr = requestedDpr;
            var _a = dimensions.current.size, width = _a.width, height = _a.height;
            var xScale = width / newSize.width;
            var yScale = height / newSize.height;
            var maxScale = Math.max(xScale, yScale);
            dpr = clamp(0.75, 4, maxScale);
            dimensions.current = {
                size: { width: newSize.width, height: newSize.height },
                dpr: dpr,
            };
            gl.setSize(newSize.width, newSize.height);
            gl.setPixelRatio(dpr);
        });
        /**
         * When a projection animation completes we want to update the camera to
         * match the recorded layout of the element.
         */
        var removeAnimationCompleteListener = projection.addEventListener("animationComplete", function () {
            var actual = (projection.layout || {}).actual;
            if (actual) {
                setTimeout(function () {
                    var newSize = calcBoxSize(actual);
                    updateCamera(newSize);
                    dimensions.current = { size: newSize };
                    gl.setSize(newSize.width, newSize.height);
                    gl.setPixelRatio(requestedDpr);
                }, 50);
            }
        });
        return function () {
            removeProjectionUpdateListener();
            removeLayoutMeasureListener();
            removeAnimationCompleteListener();
        };
    }, []);
    useLayoutEffect(function () {
        var cam = layoutCamera.current;
        if (makeDefault && cam) {
            var oldCam_1 = camera;
            set(function () { return ({ camera: cam }); });
            return function () { return set(function () { return ({ camera: oldCam_1 }); }); };
        }
    }, [camera, layoutCamera, makeDefault, set]);
    return { size: size, camera: camera, cameraRef: layoutCamera };
}

export { useLayoutCamera };
