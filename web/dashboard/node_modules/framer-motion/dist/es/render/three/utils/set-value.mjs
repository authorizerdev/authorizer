import { Vector3, Euler, Color } from 'three';

var setVector = function (name, defaultValue) {
    return function (i) {
        return function (instance, value) {
            var _a;
            (_a = instance[name]) !== null && _a !== void 0 ? _a : (instance[name] = new Vector3(defaultValue));
            var vector = instance[name];
            vector.setComponent(i, value);
        };
    };
};
var setEuler = function (name, defaultValue) {
    return function (axis) {
        return function (instance, value) {
            var _a;
            (_a = instance[name]) !== null && _a !== void 0 ? _a : (instance[name] = new Euler(defaultValue));
            var euler = instance[name];
            euler[axis] = value;
        };
    };
};
var setColor = function (name) { return function (instance, value) {
    var _a;
    (_a = instance[name]) !== null && _a !== void 0 ? _a : (instance[name] = new Color(value));
    instance[name].set(value);
}; };
var setScale = setVector("scale", 1);
var setPosition = setVector("position", 0);
var setRotation = setEuler("rotation", 0);
var setters = {
    x: setPosition(0),
    y: setPosition(1),
    z: setPosition(2),
    scale: function (instance, value) {
        var _a;
        (_a = instance.scale) !== null && _a !== void 0 ? _a : (instance.scale = new Vector3(1));
        var scale = instance.scale;
        scale.set(value, value, value);
    },
    scaleX: setScale(0),
    scaleY: setScale(1),
    scaleZ: setScale(2),
    rotateX: setRotation("x"),
    rotateY: setRotation("y"),
    rotateZ: setRotation("z"),
    color: setColor("color"),
    specular: setColor("specular"),
};
function setThreeValue(instance, key, values) {
    if (setters[key]) {
        setters[key](instance, values[key]);
    }
    else {
        instance[key] = values[key];
    }
}

export { setThreeValue };
