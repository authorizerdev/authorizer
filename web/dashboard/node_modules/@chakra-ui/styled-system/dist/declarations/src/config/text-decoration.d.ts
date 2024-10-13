import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { ResponsiveValue, Token } from "../utils";
export declare const textDecoration: Config;
export interface TextDecorationProps {
    /**
     * The CSS `text-decoration` property
     */
    textDecoration?: Token<CSS.Property.TextDecoration | number>;
    /**
     * The CSS `text-decoration` property
     */
    textDecor?: Token<CSS.Property.TextDecoration | number>;
    /**
     * The CSS `text-decoration-color` property
     */
    textDecorationColor?: Token<CSS.Property.TextDecorationColor, "colors">;
    /**
     * The CSS `text-decoration-thickness` property
     */
    textDecorationThickness?: ResponsiveValue<CSS.Property.TextDecorationThickness>;
    /**
     * The CSS `text-decoration-style` property
     */
    textDecorationStyle?: ResponsiveValue<CSS.Property.TextDecorationStyle>;
    /**
     * The CSS `text-decoration-line` property
     */
    textDecorationLine?: ResponsiveValue<CSS.Property.TextDecorationLine>;
    /**
     * The CSS `text-underline-offset` property
     */
    textUnderlineOffset?: ResponsiveValue<CSS.Property.TextUnderlineOffset>;
    /**
     * The `text-shadow` property
     */
    textShadow?: Token<CSS.Property.TextShadow | number, "shadows">;
}
//# sourceMappingURL=text-decoration.d.ts.map