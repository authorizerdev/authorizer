import { __assign } from 'tslib';
import { createElement } from 'react';
import { filterProps } from '../dom/utils/filter-props.mjs';
import { useHover } from './gestures/use-hover.mjs';
import { useTap } from './gestures/use-tap.mjs';

var useRender = function (Component, props, _projectionId, ref, _state, isStatic, visualElement) {
    return createElement(Component, __assign(__assign(__assign(__assign({ ref: ref }, filterProps(props, false, false)), { onUpdate: props.onInstanceUpdate }), useHover(isStatic, props, visualElement)), useTap(isStatic, props, visualElement)));
};

export { useRender };
