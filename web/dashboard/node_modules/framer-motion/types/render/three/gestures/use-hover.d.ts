import { MeshProps, ThreeEvent } from "@react-three/fiber";
import { VisualElement } from "../../types";
import { ThreeMotionProps } from "../types";
export declare function useHover(isStatic: boolean, { whileHover, onHoverStart, onHoverEnd, onPointerOver, onPointerOut, }: ThreeMotionProps & MeshProps, visualElement?: VisualElement): {
    onPointerOver?: undefined;
    onPointerOut?: undefined;
} | {
    onPointerOver: (event: ThreeEvent<any>) => void;
    onPointerOut: (event: ThreeEvent<any>) => void;
};
