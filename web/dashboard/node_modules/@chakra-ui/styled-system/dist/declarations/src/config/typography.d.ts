import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { ResponsiveValue, Token } from "../utils";
export declare const typography: Config;
/**
 * Types for typography related CSS properties
 */
export interface TypographyProps {
    /**
     * The CSS `font-weight` property
     */
    fontWeight?: Token<number | (string & {}), "fontWeights">;
    /**
     * The CSS `line-height` property
     */
    lineHeight?: Token<CSS.Property.LineHeight | number, "lineHeights">;
    /**
     * The CSS `letter-spacing` property
     */
    letterSpacing?: Token<CSS.Property.LetterSpacing | number, "letterSpacings">;
    /**
     * The CSS `font-size` property
     */
    fontSize?: Token<CSS.Property.FontSize | number, "fontSizes">;
    /**
     * The CSS `font-family` property
     */
    fontFamily?: Token<CSS.Property.FontFamily, "fonts">;
    /**
     * The CSS `text-align` property
     */
    textAlign?: Token<CSS.Property.TextAlign>;
    /**
     * The CSS `font-style` property
     */
    fontStyle?: Token<CSS.Property.FontStyle>;
    /**
     * The CSS `word-break` property
     */
    wordBreak?: Token<CSS.Property.WordBreak>;
    /**
     * The CSS `overflow-wrap` property
     */
    overflowWrap?: Token<CSS.Property.OverflowWrap>;
    /**
     * The CSS `text-overflow` property
     */
    textOverflow?: Token<CSS.Property.TextOverflow>;
    /**
     * The CSS `text-transform` property
     */
    textTransform?: Token<CSS.Property.TextTransform>;
    /**
     * The CSS `white-space` property
     */
    whiteSpace?: Token<CSS.Property.WhiteSpace>;
    /**
     * Used to visually truncate a text after a number of lines.
     */
    noOfLines?: ResponsiveValue<number>;
    /**
     * If `true`, it clamps truncate a text after one line.
     * @deprecated - Use `noOfLines` instead
     */
    isTruncated?: boolean;
}
//# sourceMappingURL=typography.d.ts.map