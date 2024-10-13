import { __assign } from 'tslib';
import { createMotionComponent } from '../../motion/index.mjs';
import { animations } from '../../motion/features/animations.mjs';
import { makeUseVisualState } from '../../motion/utils/use-visual-state.mjs';
import { useRender } from './use-render.mjs';
import { createRenderState, createVisualElement } from './create-visual-element.mjs';
import { scrapeMotionValuesFromProps } from './utils/scrape-motion-value.mjs';

var useVisualState = makeUseVisualState({
    scrapeMotionValuesFromProps: scrapeMotionValuesFromProps,
    createRenderState: createRenderState,
});
var preloadedFeatures = __assign({}, animations);
function custom(Component) {
    return createMotionComponent({
        Component: Component,
        preloadedFeatures: preloadedFeatures,
        useRender: useRender,
        useVisualState: useVisualState,
        createVisualElement: createVisualElement,
    });
}
var componentCache = new Map();
var motion = new Proxy(custom, {
    get: function (_, key) {
        !componentCache.has(key) && componentCache.set(key, custom(key));
        return componentCache.get(key);
    },
});

export { motion };
