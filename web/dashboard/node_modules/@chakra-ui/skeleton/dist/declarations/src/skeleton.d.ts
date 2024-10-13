import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface SkeletonOptions {
    /**
     * The color at the animation start
     */
    startColor?: string;
    /**
     * The color at the animation end
     */
    endColor?: string;
    /**
     * If `true`, it'll render its children with a nice fade transition
     */
    isLoaded?: boolean;
    /**
     * The animation speed in seconds
     * @default
     * 0.8
     */
    speed?: number;
    /**
     * The fadeIn duration in seconds
     *
     * @default
     * 0.4
     */
    fadeDuration?: number;
}
export declare type ISkeleton = SkeletonOptions;
export interface SkeletonProps extends HTMLChakraProps<"div">, SkeletonOptions, ThemingProps<"Skeleton"> {
}
export declare const Skeleton: import("@chakra-ui/system").ComponentWithAs<"div", SkeletonProps>;
export interface SkeletonTextProps extends SkeletonProps {
    spacing?: SkeletonProps["margin"];
    skeletonHeight?: SkeletonProps["height"];
    startColor?: SkeletonProps["startColor"];
    endColor?: SkeletonProps["endColor"];
    isLoaded?: SkeletonProps["isLoaded"];
}
export declare const SkeletonText: React.FC<SkeletonTextProps>;
export declare const SkeletonCircle: React.FC<SkeletonProps>;
//# sourceMappingURL=skeleton.d.ts.map