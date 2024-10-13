import { MeshProps } from "@react-three/fiber";
import { VisualElement } from "../../types";
import { ThreeMotionProps } from "../types";
export declare function useTap(isStatic: boolean, { whileTap, onTapStart, onTap, onTapCancel, onPointerDown, }: ThreeMotionProps & MeshProps, visualElement?: VisualElement): {
    onPointerDown?: undefined;
} | {
    onPointerDown: EventListener;
};
