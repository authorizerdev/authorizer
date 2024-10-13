import { __rest } from 'tslib';
import { visualElement } from '../index.mjs';
import { createBox } from '../../projection/geometry/models.mjs';
import { checkTargetForNewValues } from '../utils/setters.mjs';
import { setThreeValue } from './utils/set-value.mjs';
import { readThreeValue } from './utils/read-value.mjs';
import { scrapeMotionValuesFromProps } from './utils/scrape-motion-value.mjs';

var createRenderState = function () { return ({}); };
var threeVisualElement = visualElement({
    treeType: "three",
    readValueFromInstance: readThreeValue,
    getBaseTarget: function () {
        return undefined;
    },
    sortNodePosition: function (a, b) {
        return a.id - b.id;
    },
    makeTargetAnimatable: function (element, _a) {
        _a.transition; var target = __rest(_a, ["transition"]);
        checkTargetForNewValues(element, target, {});
        return target;
    },
    restoreTransform: function () { },
    resetTransform: function () { },
    removeValueFromRenderState: function (_key, _renderState) { },
    measureViewportBox: createBox,
    scrapeMotionValuesFromProps: scrapeMotionValuesFromProps,
    build: function (_element, state, latestValues) {
        for (var key in latestValues) {
            state[key] = latestValues[key];
        }
    },
    render: function (instance, renderState) {
        for (var key in renderState) {
            setThreeValue(instance, key, renderState);
        }
    },
});
var createVisualElement = function (_, options) {
    return threeVisualElement(options);
};

export { createRenderState, createVisualElement, threeVisualElement };
