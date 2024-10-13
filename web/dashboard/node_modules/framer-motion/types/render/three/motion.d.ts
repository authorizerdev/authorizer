/// <reference types="react" />
import type { ThreeMotionComponents } from "./types";
declare function custom<Props>(Component: string): import("react").ForwardRefExoticComponent<import("react").PropsWithoutRef<Props & import("../..").MotionProps> & import("react").RefAttributes<any>>;
export declare const motion: typeof custom & ThreeMotionComponents;
export {};
