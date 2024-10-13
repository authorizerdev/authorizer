import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
interface SpinnerOptions {
    /**
     * The color of the empty area in the spinner
     */
    emptyColor?: string;
    /**
     * The color of the spinner
     */
    color?: string;
    /**
     * The thickness of the spinner
     * @example
     * ```jsx
     * <Spinner thickness="4px"/>
     * ```
     */
    thickness?: string;
    /**
     * The speed of the spinner.
     * @example
     * ```jsx
     * <Spinner speed="0.2s"/>
     * ```
     */
    speed?: string;
    /**
     * For accessibility, it is important to add a fallback loading text.
     * This text will be visible to screen readers.
     */
    label?: string;
}
export interface SpinnerProps extends Omit<HTMLChakraProps<"div">, keyof SpinnerOptions>, SpinnerOptions, ThemingProps<"Spinner"> {
}
/**
 * Spinner is used to indicate the loading state of a page or a component,
 * It renders a `div` by default.
 *
 * @see Docs https://chakra-ui.com/spinner
 */
export declare const Spinner: import("@chakra-ui/system").ComponentWithAs<"div", SpinnerProps>;
export {};
//# sourceMappingURL=spinner.d.ts.map