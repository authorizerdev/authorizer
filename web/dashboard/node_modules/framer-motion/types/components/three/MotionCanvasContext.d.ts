import { Size } from "@react-three/fiber";
import { MutableRefObject, RefObject, Ref } from "react";
export declare type DimensionsState = {
    size: Size;
    dpr?: number;
};
export declare type SetLayoutCamera = (ref: Ref<any>) => void;
export declare type SetDimensions = (state: DimensionsState) => void;
export interface MotionCanvasContextProps {
    layoutCamera: RefObject<any>;
    dimensions: MutableRefObject<DimensionsState>;
    requestedDpr: number;
}
export declare const MotionCanvasContext: import("react").Context<MotionCanvasContextProps | undefined>;
