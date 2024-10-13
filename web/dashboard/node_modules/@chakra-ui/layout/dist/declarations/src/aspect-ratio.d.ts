import { ResponsiveValue, HTMLChakraProps } from "@chakra-ui/system";
interface AspectRatioOptions {
    /**
     * The aspect ratio of the Box. Common values are:
     *
     * `21/9`, `16/9`, `9/16`, `4/3`, `1.85/1`
     */
    ratio?: ResponsiveValue<number>;
}
export interface AspectRatioProps extends HTMLChakraProps<"div">, AspectRatioOptions {
}
/**
 * React component used to cropping media (videos, images and maps)
 * to a desired aspect ratio.
 *
 * @see Docs https://chakra-ui.com/aspectratiobox
 */
export declare const AspectRatio: import("@chakra-ui/system").ComponentWithAs<"div", AspectRatioProps>;
export {};
//# sourceMappingURL=aspect-ratio.d.ts.map