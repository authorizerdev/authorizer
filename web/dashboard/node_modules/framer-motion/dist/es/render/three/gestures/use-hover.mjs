import { AnimationType } from '../../utils/types.mjs';

function useHover(isStatic, _a, visualElement) {
    var whileHover = _a.whileHover, onHoverStart = _a.onHoverStart, onHoverEnd = _a.onHoverEnd, onPointerOver = _a.onPointerOver, onPointerOut = _a.onPointerOut;
    var isHoverEnabled = whileHover || onHoverStart || onHoverEnd;
    if (isStatic || !visualElement || !isHoverEnabled)
        return {};
    return {
        onPointerOver: function (event) {
            var _a;
            (_a = visualElement.animationState) === null || _a === void 0 ? void 0 : _a.setActive(AnimationType.Hover, true);
            onPointerOver === null || onPointerOver === void 0 ? void 0 : onPointerOver(event);
        },
        onPointerOut: function (event) {
            var _a;
            (_a = visualElement.animationState) === null || _a === void 0 ? void 0 : _a.setActive(AnimationType.Hover, false);
            onPointerOut === null || onPointerOut === void 0 ? void 0 : onPointerOut(event);
        },
    };
}

export { useHover };
