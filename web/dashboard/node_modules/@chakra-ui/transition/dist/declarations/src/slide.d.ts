import { HTMLMotionProps } from "framer-motion";
import * as React from "react";
import { SlideDirection, WithTransitionConfig } from "./transition-utils";
export type { SlideDirection };
export interface SlideOptions {
    /**
     * The direction to slide from
     * @default "right"
     */
    direction?: SlideDirection;
}
export interface SlideProps extends WithTransitionConfig<HTMLMotionProps<"div">>, SlideOptions {
}
export declare const Slide: React.ForwardRefExoticComponent<SlideProps & React.RefAttributes<HTMLDivElement>>;
//# sourceMappingURL=slide.d.ts.map