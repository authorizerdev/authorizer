/// <reference types="react" />
import { SystemStyleObject } from "@chakra-ui/system";
import { Dict, runIfFn } from "@chakra-ui/utils";
export interface StyleConfig {
    baseStyle?: SystemStyleObject;
    sizes?: {
        [size: string]: SystemStyleObject;
    };
    variants?: {
        [variant: string]: SystemStyleObject;
    };
    defaultProps?: {
        size?: string;
        variant?: string;
        colorScheme?: string;
    };
}
declare type Anatomy = {
    __type: string;
};
export interface MultiStyleConfig<T extends Anatomy = Anatomy> {
    baseStyle?: PartsStyleObject<T>;
    sizes?: {
        [size: string]: PartsStyleObject<T> | PartsStyleFunction<T>;
    };
    variants?: {
        [variant: string]: PartsStyleObject<T> | PartsStyleFunction<T>;
    };
    defaultProps?: StyleConfig["defaultProps"];
}
export type { SystemStyleObject };
export declare type StyleFunctionProps = {
    colorScheme: string;
    colorMode: "light" | "dark";
    orientation?: "horizontal" | "vertical";
    theme: Dict;
    [key: string]: any;
};
export declare type SystemStyleFunction = (props: StyleFunctionProps) => SystemStyleObject;
export declare type SystemStyleInterpolation = SystemStyleObject | SystemStyleFunction;
export declare type PartsStyleObject<T extends Anatomy = Anatomy> = Partial<Record<T["__type"], SystemStyleObject>>;
export declare type PartsStyleFunction<T extends Anatomy = Anatomy> = (props: StyleFunctionProps) => PartsStyleObject<T>;
export declare type PartsStyleInterpolation<T extends Anatomy = Anatomy> = PartsStyleObject<T> | PartsStyleFunction<T>;
export declare type GlobalStyleProps = StyleFunctionProps;
export declare type GlobalStyles = {
    global?: SystemStyleInterpolation;
};
export declare type JSXElementStyles = {
    [K in keyof JSX.IntrinsicElements]?: SystemStyleObject;
};
export { runIfFn };
export declare type Styles = GlobalStyles & JSXElementStyles;
export declare function mode(light: any, dark: any): (props: Dict | StyleFunctionProps) => any;
export declare function orient(options: {
    orientation?: "vertical" | "horizontal";
    vertical: SystemStyleObject;
    horizontal: SystemStyleObject;
}): import("@chakra-ui/system").CSSObject;
//# sourceMappingURL=component.d.ts.map