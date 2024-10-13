import { HTMLMotionProps } from "framer-motion";
import * as React from "react";
import { WithTransitionConfig } from "./transition-utils";
export interface FadeProps extends WithTransitionConfig<HTMLMotionProps<"div">> {
}
export declare const fadeConfig: HTMLMotionProps<"div">;
export declare const Fade: React.ForwardRefExoticComponent<FadeProps & React.RefAttributes<HTMLDivElement>>;
//# sourceMappingURL=fade.d.ts.map