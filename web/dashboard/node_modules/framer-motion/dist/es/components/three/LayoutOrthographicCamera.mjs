import { __assign } from 'tslib';
import * as React from 'react';
import mergeRefs from 'react-merge-refs';
import { motion } from '../../render/three/motion.mjs';
import { useLayoutCamera } from './use-layout-camera.mjs';

var LayoutOrthographicCamera = React.forwardRef(function (props, ref) {
    var _a = useLayoutCamera(props, function (newSize) {
        var cam = cameraRef.current;
        if (cam) {
            cam.left = newSize.width / -2;
            cam.right = newSize.width / 2;
            cam.top = newSize.height / 2;
            cam.bottom = newSize.height / -2;
            cam.updateProjectionMatrix();
        }
    }), size = _a.size, cameraRef = _a.cameraRef;
    return (React.createElement(motion.orthographicCamera, __assign({ left: size.width / -2, right: size.width / 2, top: size.height / 2, bottom: size.height / -2, ref: mergeRefs([cameraRef, ref]) }, props)));
});

export { LayoutOrthographicCamera };
