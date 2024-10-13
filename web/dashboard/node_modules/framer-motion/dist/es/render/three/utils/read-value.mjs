import { Color } from 'three';

var readVector = function (name, defaultValue) {
    return function (axis) {
        return function (instance) {
            var value = instance[name];
            return value ? value[axis] : defaultValue;
        };
    };
};
var readPosition = readVector("position", 0);
var readScale = readVector("scale", 1);
var readRotation = readVector("rotation", 0);
var readers = {
    x: readPosition("x"),
    y: readPosition("y"),
    z: readPosition("z"),
    scale: readScale("x"),
    scaleX: readScale("x"),
    scaleY: readScale("y"),
    scaleZ: readScale("z"),
    rotateX: readRotation("x"),
    rotateY: readRotation("y"),
    rotateZ: readRotation("z"),
};
function readAnimatableValue(value) {
    if (value === undefined) {
        return;
    }
    else if (value instanceof Color) {
        return value.getStyle();
    }
    else {
        return value;
    }
}
function readThreeValue(instance, name) {
    var _a;
    return readers[name]
        ? readers[name](instance)
        : (_a = readAnimatableValue(instance[name])) !== null && _a !== void 0 ? _a : 0;
}

export { readThreeValue };
