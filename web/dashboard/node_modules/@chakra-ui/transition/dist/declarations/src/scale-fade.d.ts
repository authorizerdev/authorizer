import { HTMLMotionProps } from "framer-motion";
import * as React from "react";
import { WithTransitionConfig } from "./transition-utils";
interface ScaleFadeOptions {
    /**
     * The initial scale of the element
     * @default 0.95
     */
    initialScale?: number;
    /**
     * If `true`, the element will transition back to exit state
     */
    reverse?: boolean;
}
export declare const scaleFadeConfig: HTMLMotionProps<"div">;
export interface ScaleFadeProps extends ScaleFadeOptions, WithTransitionConfig<HTMLMotionProps<"div">> {
}
export declare const ScaleFade: React.ForwardRefExoticComponent<ScaleFadeProps & React.RefAttributes<HTMLDivElement>>;
export {};
//# sourceMappingURL=scale-fade.d.ts.map