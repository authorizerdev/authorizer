import { AnalyzeBreakpointsReturn, Dict } from "@chakra-ui/utils";
import { ThemeTypings } from "../theming.types";
export declare type ResponsiveArray<T> = Array<T | null>;
export declare type ResponsiveObject<T> = Partial<Record<ThemeTypings["breakpoints"] | string, T>>;
export declare type ResponsiveValue<T> = T | ResponsiveArray<T> | ResponsiveObject<T>;
export declare type Length = string | 0 | number;
export declare type Union<T> = T | (string & {});
export declare type Token<CSSType, ThemeKey = unknown> = ThemeKey extends keyof ThemeTypings ? ResponsiveValue<Union<CSSType | ThemeTypings[ThemeKey]>> : ResponsiveValue<CSSType>;
export declare type CSSMap = Dict<{
    value: string;
    var: string;
    varRef: string;
}>;
export declare type Transform = (value: any, theme: CssTheme, styles?: Dict) => any;
export declare type WithCSSVar<T> = T & {
    __cssVars: Dict;
    __cssMap: CSSMap;
    __breakpoints: AnalyzeBreakpointsReturn;
};
export declare type CssTheme = WithCSSVar<{
    breakpoints: Dict;
    direction?: "ltr" | "rtl";
    [key: string]: any;
}>;
//# sourceMappingURL=types.d.ts.map