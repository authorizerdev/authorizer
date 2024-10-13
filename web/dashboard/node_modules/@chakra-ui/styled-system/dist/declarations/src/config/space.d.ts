import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { Token } from "../utils";
export declare const space: Config;
/**
 * Types for space related CSS properties
 */
export interface SpaceProps {
    /**
     * Margin on top, left, bottom and right
     */
    m?: Token<CSS.Property.Margin | number, "space">;
    /**
     * Margin on top, left, bottom and right
     */
    margin?: Token<CSS.Property.Margin | number, "space">;
    /**
     * Margin on top
     */
    mt?: Token<CSS.Property.Margin | number, "space">;
    marginBlockStart?: Token<CSS.Property.MarginBlockStart | number, "space">;
    /**
     * Margin on top
     */
    marginTop?: Token<CSS.Property.MarginTop | number, "space">;
    /**
     * Margin on right
     */
    mr?: Token<CSS.Property.MarginRight | number, "space">;
    /**
     * When direction is `ltr`, `marginInlineEnd` is equivalent to `marginRight`.
     * When direction is `rtl`, `marginInlineEnd` is equivalent to `marginLeft`.
     */
    marginInlineEnd?: Token<CSS.Property.MarginInlineEnd | number, "space">;
    /**
     * When direction is `ltr`, `marginEnd` is equivalent to `marginRight`.
     * When direction is `rtl`, `marginEnd` is equivalent to `marginLeft`.
     */
    marginEnd?: Token<CSS.Property.MarginInlineEnd | number, "space">;
    /**
     * When direction is `ltr`, `me` is equivalent to `marginRight`.
     * When direction is `rtl`, `me` is equivalent to `marginLeft`.
     */
    me?: Token<CSS.Property.MarginInlineEnd | number, "space">;
    /**
     * Margin on right
     */
    marginRight?: Token<CSS.Property.MarginRight | number, "space">;
    /**
     * Margin on bottom
     */
    mb?: Token<CSS.Property.MarginBottom | number, "space">;
    marginBlockEnd?: Token<CSS.Property.MarginBlockEnd | number, "space">;
    /**
     * Margin on bottom
     */
    marginBottom?: Token<CSS.Property.MarginBottom | number, "space">;
    /**
     * Margin on left
     */
    ml?: Token<CSS.Property.MarginLeft | number, "space">;
    /**
     * When direction is `ltr`, `marginInlineStart` is equivalent to `marginLeft`.
     * When direction is `rtl`, `marginInlineStart` is equivalent to `marginRight`.
     */
    marginInlineStart?: Token<CSS.Property.MarginInlineStart | number, "space">;
    /**
     * When direction is `ltr`, `marginStart` is equivalent to `marginLeft`.
     * When direction is `rtl`, `marginStart` is equivalent to `marginRight`.
     */
    marginStart?: Token<CSS.Property.MarginInlineStart | number, "space">;
    /**
     * When direction is `ltr`, `ms` is equivalent to `marginLeft`.
     * When direction is `rtl`, `ms` is equivalent to `marginRight`.
     */
    ms?: Token<CSS.Property.MarginInlineStart | number, "space">;
    /**
     * Margin on left
     */
    marginLeft?: Token<CSS.Property.MarginLeft | number, "space">;
    /**
     * Margin on left and right
     */
    mx?: Token<CSS.Property.Margin | number, "space">;
    marginInline?: Token<CSS.Property.MarginInline | number, "space">;
    /**
     * Margin on left and right
     */
    marginX?: Token<CSS.Property.Margin | number, "space">;
    /**
     * Margin on top and bottom
     */
    my?: Token<CSS.Property.Margin | number, "space">;
    marginBlock?: Token<CSS.Property.MarginBlock | number, "space">;
    /**
     * Margin on top and bottom
     */
    marginY?: Token<CSS.Property.Margin | number, "space">;
    /**
     * Padding on top, left, bottom and right
     */
    p?: Token<CSS.Property.Padding | number, "space">;
    /**
     * Padding on top, left, bottom and right
     */
    padding?: Token<CSS.Property.Padding | number, "space">;
    /**
     * Padding on top
     */
    pt?: Token<CSS.Property.PaddingTop | number, "space">;
    paddingBlockStart?: Token<CSS.Property.PaddingBlockStart | number, "space">;
    /**
     * Padding on top
     */
    paddingTop?: Token<CSS.Property.PaddingTop | number, "space">;
    /**
     * Padding on right
     */
    pr?: Token<CSS.Property.PaddingRight | number, "space">;
    /**
     * When direction is `ltr`, `paddingInlineEnd` is equivalent to `paddingRight`.
     * When direction is `rtl`, `paddingInlineEnd` is equivalent to `paddingLeft`.
     */
    paddingInlineEnd?: Token<CSS.Property.PaddingInlineEnd | number, "space">;
    /**
     * When direction is `ltr`, `paddingEnd` is equivalent to `paddingRight`.
     * When direction is `rtl`, `paddingEnd` is equivalent to `paddingLeft`.
     */
    paddingEnd?: Token<CSS.Property.PaddingInlineEnd | number, "space">;
    /**
     * When direction is `ltr`, `pe` is equivalent to `paddingRight`.
     * When direction is `rtl`, `pe` is equivalent to `paddingLeft`.
     */
    pe?: Token<CSS.Property.PaddingInlineEnd | number, "space">;
    /**
     * Padding on right
     */
    paddingRight?: Token<CSS.Property.PaddingRight | number, "space">;
    /**
     * Padding on bottom
     */
    pb?: Token<CSS.Property.PaddingBottom | number, "space">;
    paddingBlockEnd?: Token<CSS.Property.PaddingBlockEnd | number, "space">;
    /**
     * Padding on bottom
     */
    paddingBottom?: Token<CSS.Property.PaddingBottom | number, "space">;
    /**
     * Padding on left
     */
    pl?: Token<CSS.Property.PaddingLeft | number, "space">;
    /**
     * When direction is `ltr`, `paddingInlineStart` is equivalent to `paddingLeft`.
     * When direction is `rtl`, `paddingInlineStart` is equivalent to `paddingRight`.
     */
    paddingInlineStart?: Token<CSS.Property.PaddingInlineStart | number, "space">;
    /**
     * When direction is `ltr`, `paddingStart` is equivalent to `paddingLeft`.
     * When direction is `rtl`, `paddingStart` is equivalent to `paddingRight`.
     */
    paddingStart?: Token<CSS.Property.PaddingInlineStart | number, "space">;
    /**
     * When direction is `ltr`, `ps` is equivalent to `paddingLeft`.
     * When direction is `rtl`, `ps` is equivalent to `paddingRight`.
     */
    ps?: Token<CSS.Property.PaddingInlineStart | number, "space">;
    /**
     * Padding on left
     */
    paddingLeft?: Token<CSS.Property.PaddingLeft | number, "space">;
    /**
     * Padding on left and right
     */
    px?: Token<CSS.Property.Padding | number, "space">;
    paddingInline?: Token<CSS.Property.PaddingInline | number, "space">;
    /**
     * Padding on left and right
     */
    paddingX?: Token<CSS.Property.Padding | number, "space">;
    /**
     * Padding on top and bottom
     */
    py?: Token<CSS.Property.Padding | number, "space">;
    paddingBlock?: Token<CSS.Property.PaddingBlock | number, "space">;
    /**
     * Padding on top and bottom
     */
    paddingY?: Token<CSS.Property.Padding | number, "space">;
}
//# sourceMappingURL=space.d.ts.map