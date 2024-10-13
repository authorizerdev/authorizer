import { HTMLChakraProps } from "@chakra-ui/system";
import { StringOrNumber } from "@chakra-ui/utils";
import * as React from "react";
interface CircularProgressOptions {
    /**
     * The size of the circular progress in CSS units
     */
    size?: StringOrNumber;
    /**
     * Maximum value defining 100% progress made (must be higher than 'min')
     */
    max?: number;
    /**
     * Minimum value defining 'no progress' (must be lower than 'max')
     */
    min?: number;
    /**
     * This defines the stroke width of the svg circle.
     */
    thickness?: StringOrNumber;
    /**
     * Current progress (must be between min/max)
     */
    value?: number;
    /**
     * If `true`, the cap of the progress indicator will be rounded.
     */
    capIsRound?: boolean;
    /**
     * The content of the circular progress bar. If passed, the content will be inside and centered in the progress bar.
     */
    children?: React.ReactNode;
    /**
     * The color name of the progress track. Use a color key in the theme object
     */
    trackColor?: string;
    /**
     * The color of the progress indicator. Use a color key in the theme object
     */
    color?: string;
    /**
     * The desired valueText to use in place of the value
     */
    valueText?: string;
    /**
     * A function that returns the desired valueText to use in place of the value
     */
    getValueText?(value: number, percent: number): string;
    /**
     * If `true`, the progress will be indeterminate and the `value`
     * prop will be ignored
     */
    isIndeterminate?: boolean;
}
export interface CircularProgressProps extends Omit<HTMLChakraProps<"div">, "color">, CircularProgressOptions {
}
/**
 * CircularProgress is used to indicate the progress of an activity.
 * It is built using `svg` and `circle` components with support for
 * theming and `indeterminate` state
 *
 * @see Docs https://chakra-ui.com/circularprogress
 * @todo add theming support for circular progress
 */
export declare const CircularProgress: React.FC<CircularProgressProps>;
/**
 * CircularProgress component label. In most cases it is a numeric indicator
 * of the circular progress component's value
 */
export declare const CircularProgressLabel: import("@chakra-ui/system").ChakraComponent<"div", {}>;
export interface CircularProgressLabelProps extends HTMLChakraProps<"div"> {
}
export {};
//# sourceMappingURL=circular-progress.d.ts.map