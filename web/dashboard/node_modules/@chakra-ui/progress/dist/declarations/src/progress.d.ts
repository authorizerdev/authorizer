import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
import { GetProgressPropsOptions } from "./progress.utils";
export interface ProgressLabelProps extends HTMLChakraProps<"div"> {
}
/**
 * ProgressLabel is used to show the numeric value of the progress.
 * @see Docs https://chakra-ui.com/progress
 */
export declare const ProgressLabel: React.FC<ProgressLabelProps>;
export interface ProgressFilledTrackProps extends HTMLChakraProps<"div">, GetProgressPropsOptions {
}
export interface ProgressTrackProps extends HTMLChakraProps<"div"> {
}
interface ProgressOptions {
    /**
     * The `value` of the progress indicator.
     * If `undefined` the progress bar will be in `indeterminate` state
     */
    value?: number;
    /**
     * The minimum value of the progress
     */
    min?: number;
    /**
     * The maximum value of the progress
     */
    max?: number;
    /**
     * If `true`, the progress bar will show stripe
     */
    hasStripe?: boolean;
    /**
     * If `true`, and hasStripe is `true`, the stripes will be animated
     */
    isAnimated?: boolean;
    /**
     * If `true`, the progress will be indeterminate and the `value`
     * prop will be ignored
     */
    isIndeterminate?: boolean;
}
export interface ProgressProps extends ProgressOptions, ThemingProps<"Progress">, HTMLChakraProps<"div"> {
}
/**
 * Progress (Linear)
 *
 * Progress is used to display the progress status for a task that takes a long
 * time or consists of several steps.
 *
 * It includes accessible attributes to help assistive technologies understand
 * and speak the progress values.
 *
 * @see Docs https://chakra-ui.com/progress
 */
export declare const Progress: React.FC<ProgressProps>;
export {};
//# sourceMappingURL=progress.d.ts.map