import { __assign } from 'tslib';
import * as React from 'react';
import mergeRefs from 'react-merge-refs';
import { motion } from '../../render/three/motion.mjs';
import { useLayoutCamera } from './use-layout-camera.mjs';

/**
 * Adapted from https://github.com/pmndrs/drei/blob/master/src/core/PerspectiveCamera.tsx
 */
var LayoutCamera = React.forwardRef(function (props, ref) {
    var cameraRef = useLayoutCamera(props, function (size) {
        var cam = cameraRef.current;
        if (cam && !props.manual) {
            cam.aspect = size.width / size.height;
            cam.updateProjectionMatrix();
        }
    }).cameraRef;
    return (React.createElement(motion.perspectiveCamera, __assign({ ref: mergeRefs([cameraRef, ref]) }, props)));
});

export { LayoutCamera };
