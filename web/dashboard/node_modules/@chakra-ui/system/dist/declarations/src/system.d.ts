import { StyleProps, SystemStyleObject } from "@chakra-ui/styled-system";
import { CSSObject, FunctionInterpolation } from "@emotion/styled";
import { As, ChakraComponent, ChakraProps, PropsOf } from "./system.types";
import { DOMElements } from "./system.utils";
declare type StyleResolverProps = SystemStyleObject & {
    __css?: SystemStyleObject;
    sx?: SystemStyleObject;
    theme: any;
    css?: CSSObject;
};
interface GetStyleObject {
    (options: {
        baseStyle?: SystemStyleObject | ((props: StyleResolverProps) => SystemStyleObject);
    }): FunctionInterpolation<StyleResolverProps>;
}
/**
 * Style resolver function that manages how style props are merged
 * in combination with other possible ways of defining styles.
 *
 * For example, take a component defined this way:
 * ```jsx
 * <Box fontSize="24px" sx={{ fontSize: "40px" }}></Box>
 * ```
 *
 * We want to manage the priority of the styles properly to prevent unwanted
 * behaviors. Right now, the `sx` prop has the highest priority so the resolved
 * fontSize will be `40px`
 */
export declare const toCSSObject: GetStyleObject;
interface StyledOptions {
    shouldForwardProp?(prop: string): boolean;
    label?: string;
    baseStyle?: SystemStyleObject | ((props: StyleResolverProps) => SystemStyleObject);
}
export declare function styled<T extends As, P = {}>(component: T, options?: StyledOptions): ChakraComponent<T, P>;
export declare type HTMLChakraComponents = {
    [Tag in DOMElements]: ChakraComponent<Tag, {}>;
};
export declare type HTMLChakraProps<T extends As> = Omit<PropsOf<T>, T extends "svg" ? "ref" | "children" | keyof StyleProps : "ref" | keyof StyleProps> & ChakraProps & {
    as?: As;
};
declare type ChakraFactory = {
    <T extends As, P = {}>(component: T, options?: StyledOptions): ChakraComponent<T, P>;
};
export declare const chakra: ChakraFactory & HTMLChakraComponents;
export {};
//# sourceMappingURL=system.d.ts.map