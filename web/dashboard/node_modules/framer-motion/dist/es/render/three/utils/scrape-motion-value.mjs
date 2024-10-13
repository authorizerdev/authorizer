import { isMotionValue } from '../../../value/utils/is-motion-value.mjs';

var axes = ["x", "y", "z"];
var valueMap = {
    "position-x": "x",
    "position-y": "y",
    "position-z": "z",
    "rotation-x": "rotateX",
    "rotation-y": "rotateY",
    "rotation-z": "rotateZ",
    "scale-x": "scaleX",
    "scale-y": "scaleY",
    "scale-z": "scaleZ",
};
var scrapeMotionValuesFromProps = function (props) {
    var motionValues = {};
    for (var key in props) {
        var prop = props[key];
        if (isMotionValue(prop)) {
            motionValues[valueMap[key] || key] = prop;
        }
        else if (Array.isArray(prop)) {
            for (var i = 0; i < prop.length; i++) {
                var value = prop[i];
                if (isMotionValue(value)) {
                    var name_1 = valueMap[key + "-" + axes[i]];
                    motionValues[name_1] = value;
                }
            }
        }
    }
    return motionValues;
};

export { scrapeMotionValuesFromProps };
