import * as React from "react";
import { OrthographicCamera as OrthographicCameraImpl } from "three";
import { LayoutCameraProps } from "./types";
import { ThreeMotionProps } from "../../render/three/types";
export declare const LayoutOrthographicCamera: React.ForwardRefExoticComponent<Pick<Omit<import("@react-three/fiber").Node<OrthographicCameraImpl, typeof OrthographicCameraImpl>, import("@react-three/fiber").NonFunctionKeys<{
    position?: import("@react-three/fiber").Vector3 | undefined;
    up?: import("@react-three/fiber").Vector3 | undefined;
    scale?: import("@react-three/fiber").Vector3 | undefined;
    rotation?: import("@react-three/fiber").Euler | undefined;
    matrix?: import("@react-three/fiber").Matrix4 | undefined;
    quaternion?: import("@react-three/fiber").Quaternion | undefined;
    layers?: import("@react-three/fiber").Layers | undefined;
    dispose?: (() => void) | null | undefined;
}>> & {
    position?: import("@react-three/fiber").Vector3 | undefined;
    up?: import("@react-three/fiber").Vector3 | undefined;
    scale?: import("@react-three/fiber").Vector3 | undefined;
    rotation?: import("@react-three/fiber").Euler | undefined;
    matrix?: import("@react-three/fiber").Matrix4 | undefined;
    quaternion?: import("@react-three/fiber").Quaternion | undefined;
    layers?: import("@react-three/fiber").Layers | undefined;
    dispose?: (() => void) | null | undefined;
} & import("@react-three/fiber/dist/declarations/src/core/events").EventHandlers & LayoutCameraProps & ThreeMotionProps, "transition" | "scale" | "bottom" | "clear" | "left" | "position" | "right" | "top" | "zoom" | "copy" | "visible" | "add" | "type" | "onUpdate" | "rotateX" | "rotateY" | "key" | "id" | "name" | "children" | "onClick" | "onContextMenu" | "onDoubleClick" | "onPointerDown" | "onPointerMove" | "onPointerUp" | "onPointerCancel" | "onPointerEnter" | "onPointerLeave" | "onPointerOver" | "onPointerOut" | "onWheel" | "onAnimationStart" | "translateX" | "translateY" | "translateZ" | "rotateZ" | "clone" | "manual" | "onAnimationComplete" | "onUnmount" | "animate" | "exit" | "variants" | "onTap" | "onTapStart" | "onTapCancel" | "whileTap" | "whileHover" | "onHoverStart" | "onHoverEnd" | "view" | "quaternion" | "parent" | "attach" | "onInstanceUpdate" | "attachArray" | "attachObject" | "attachFns" | "args" | "up" | "rotation" | "matrix" | "layers" | "dispose" | "raycast" | "uuid" | "modelViewMatrix" | "normalMatrix" | "matrixWorld" | "matrixAutoUpdate" | "matrixWorldNeedsUpdate" | "castShadow" | "receiveShadow" | "frustumCulled" | "renderOrder" | "animations" | "userData" | "customDepthMaterial" | "customDistanceMaterial" | "isObject3D" | "onBeforeRender" | "onAfterRender" | "applyMatrix4" | "applyQuaternion" | "setRotationFromAxisAngle" | "setRotationFromEuler" | "setRotationFromMatrix" | "setRotationFromQuaternion" | "rotateOnAxis" | "rotateOnWorldAxis" | "translateOnAxis" | "localToWorld" | "worldToLocal" | "lookAt" | "remove" | "removeFromParent" | "getObjectById" | "getObjectByName" | "getObjectByProperty" | "getWorldPosition" | "getWorldQuaternion" | "getWorldScale" | "getWorldDirection" | "traverse" | "traverseVisible" | "traverseAncestors" | "updateMatrix" | "updateMatrixWorld" | "updateWorldMatrix" | "toJSON" | "addEventListener" | "hasEventListener" | "removeEventListener" | "dispatchEvent" | "onPointerMissed" | "matrixWorldInverse" | "projectionMatrix" | "projectionMatrixInverse" | "isCamera" | "near" | "far" | "setViewOffset" | "clearViewOffset" | "updateProjectionMatrix" | "isOrthographicCamera" | "makeDefault"> & React.RefAttributes<unknown>>;
