import { Dict } from "@chakra-ui/utils";
import * as CSS from "csstype";
import type { BackgroundProps, BorderProps, ColorProps, EffectProps, FilterProps, FlexboxProps, GridProps, InteractivityProps, LayoutProps, ListProps, OtherProps, PositionProps, RingProps, SpaceProps, TextDecorationProps, TransformProps, TransitionProps, TypographyProps } from "./config";
import { Pseudos } from "./pseudos";
import { ResponsiveValue } from "./utils/types";
export interface StyleProps extends SpaceProps, ColorProps, TransitionProps, TypographyProps, FlexboxProps, TransformProps, GridProps, FilterProps, LayoutProps, BorderProps, EffectProps, BackgroundProps, ListProps, PositionProps, RingProps, InteractivityProps, TextDecorationProps, OtherProps {
}
export interface SystemCSSProperties extends CSS.Properties, Omit<StyleProps, keyof CSS.Properties> {
}
export declare type ThemeThunk<T> = T | ((theme: Dict) => T);
declare type PropertyValue<K extends keyof SystemCSSProperties> = ThemeThunk<ResponsiveValue<boolean | number | string | SystemCSSProperties[K]>>;
export declare type CSSWithMultiValues = {
    [K in keyof SystemCSSProperties]?: K extends keyof StyleProps ? StyleProps[K] | PropertyValue<K> : PropertyValue<K>;
};
declare type PseudoKeys = keyof CSS.Pseudos | keyof Pseudos;
declare type PseudoSelectorDefinition<D> = D | RecursivePseudo<D>;
export declare type RecursivePseudo<D> = {
    [K in PseudoKeys]?: PseudoSelectorDefinition<D> & D;
};
declare type CSSDefinition<D> = D | string | RecursiveCSSSelector<D | string>;
export interface RecursiveCSSSelector<D> {
    [selector: string]: CSSDefinition<D> & D;
}
export declare type RecursiveCSSObject<D> = D & (D | RecursivePseudo<D> | RecursiveCSSSelector<D>);
export declare type CSSObject = RecursiveCSSObject<CSSWithMultiValues>;
export declare type SystemStyleObject = CSSObject;
export interface FunctionCSSInterpolation {
    (theme: Dict): CSSObject;
}
export declare type StyleObjectOrFn = CSSObject | FunctionCSSInterpolation;
declare type PseudoProps = {
    [K in keyof Pseudos]?: SystemStyleObject;
};
export interface SystemProps extends StyleProps, PseudoProps {
}
export {};
//# sourceMappingURL=system.types.d.ts.map