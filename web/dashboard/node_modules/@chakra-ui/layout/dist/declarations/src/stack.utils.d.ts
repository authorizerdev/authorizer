import { ResponsiveValue, SystemProps } from "@chakra-ui/system";
export declare type StackDirection = ResponsiveValue<"row" | "column" | "row-reverse" | "column-reverse">;
/**
 * If we ever run into SSR issues with this, check this post to find a fix for it:
 * @see https://medium.com/@emmenko/patching-lobotomized-owl-selector-for-emotion-ssr-5a582a3c424c
 */
export declare const selector = "& > *:not(style) ~ *:not(style)";
interface Options {
    spacing: SystemProps["margin"];
    direction: StackDirection;
}
export declare function getStackStyles(options: Options): {
    flexDirection: StackDirection;
    "& > *:not(style) ~ *:not(style)": any;
};
export declare function getDividerStyles(options: Options): {
    "&": any;
};
export {};
//# sourceMappingURL=stack.utils.d.ts.map