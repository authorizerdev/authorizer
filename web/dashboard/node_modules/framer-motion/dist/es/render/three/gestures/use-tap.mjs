import { pipe } from 'popmotion';
import { useRef } from 'react';
import { wrapHandler } from '../../../events/event-info.mjs';
import { addPointerEvent } from '../../../events/use-pointer-event.mjs';
import { isDragActive } from '../../../gestures/drag/utils/lock.mjs';
import { AnimationType } from '../../utils/types.mjs';

function useTap(isStatic, _a, visualElement) {
    var whileTap = _a.whileTap, onTapStart = _a.onTapStart, onTap = _a.onTap, onTapCancel = _a.onTapCancel, onPointerDown = _a.onPointerDown;
    var isTapEnabled = onTap || onTapStart || onTapCancel || whileTap;
    var isPressing = useRef(false);
    var cancelPointerEndListeners = useRef(null);
    if (isStatic || !visualElement || !isTapEnabled)
        return {};
    function removePointerEndListener() {
        var _a;
        (_a = cancelPointerEndListeners.current) === null || _a === void 0 ? void 0 : _a.call(cancelPointerEndListeners);
        cancelPointerEndListeners.current = null;
    }
    function checkPointerEnd() {
        var _a;
        removePointerEndListener();
        isPressing.current = false;
        (_a = visualElement.animationState) === null || _a === void 0 ? void 0 : _a.setActive(AnimationType.Tap, false);
        return !isDragActive();
    }
    function onPointerUp(event, info) {
        if (!checkPointerEnd())
            return;
        /**
         * We only count this as a tap gesture if the event.target is the same
         * as, or a child of, this component's element
         */
        onTap === null || onTap === void 0 ? void 0 : onTap(event, info);
    }
    function onPointerCancel(event, info) {
        if (!checkPointerEnd())
            return;
        onTapCancel === null || onTapCancel === void 0 ? void 0 : onTapCancel(event, info);
    }
    return {
        onPointerDown: wrapHandler(function (event, info) {
            var _a;
            removePointerEndListener();
            if (isPressing.current)
                return;
            isPressing.current = true;
            cancelPointerEndListeners.current = pipe(addPointerEvent(window, "pointerup", onPointerUp), addPointerEvent(window, "pointercancel", onPointerCancel));
            (_a = visualElement.animationState) === null || _a === void 0 ? void 0 : _a.setActive(AnimationType.Tap, true);
            onPointerDown === null || onPointerDown === void 0 ? void 0 : onPointerDown(event);
            onTapStart === null || onTapStart === void 0 ? void 0 : onTapStart(event, info);
        }, true),
    };
}

export { useTap };
