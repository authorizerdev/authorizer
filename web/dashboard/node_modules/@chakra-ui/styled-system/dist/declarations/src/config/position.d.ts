import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { Token } from "../utils";
export declare const position: Config;
/**
 * Types for position CSS properties
 */
export interface PositionProps {
    /**
     * The CSS `z-index` property
     */
    zIndex?: Token<CSS.Property.ZIndex, "zIndices">;
    /**
     * The CSS `top` property
     */
    top?: Token<CSS.Property.Top | number, "sizes">;
    insetBlockStart?: Token<CSS.Property.InsetBlockStart | number, "sizes">;
    /**
     * The CSS `right` property
     */
    right?: Token<CSS.Property.Right | number, "sizes">;
    /**
     * When the direction is `ltr`, `insetInlineEnd` is equivalent to `right`.
     * When the direction is `rtl`, `insetInlineEnd` is equivalent to `left`.
     */
    insetInlineEnd?: Token<CSS.Property.InsetInlineEnd | number, "sizes">;
    /**
     * When the direction is `ltr`, `insetEnd` is equivalent to `right`.
     * When the direction is `rtl`, `insetEnd` is equivalent to `left`.
     */
    insetEnd?: Token<CSS.Property.InsetInlineEnd | number, "sizes">;
    /**
     * The CSS `bottom` property
     */
    bottom?: Token<CSS.Property.Bottom | number, "sizes">;
    insetBlockEnd?: Token<CSS.Property.InsetBlockEnd | number, "sizes">;
    /**
     * The CSS `left` property
     */
    left?: Token<CSS.Property.Left | number, "sizes">;
    insetInlineStart?: Token<CSS.Property.InsetInlineStart | number, "sizes">;
    /**
     * When the direction is `start`, `end` is equivalent to `left`.
     * When the direction is `start`, `end` is equivalent to `right`.
     */
    insetStart?: Token<CSS.Property.InsetInlineStart | number, "sizes">;
    /**
     * The CSS `left`, `right`, `top`, `bottom` property
     */
    inset?: Token<CSS.Property.Inset | number, "sizes">;
    /**
     * The CSS `left`, and `right` property
     */
    insetX?: Token<CSS.Property.Inset | number, "sizes">;
    /**
     * The CSS `top`, and `bottom` property
     */
    insetY?: Token<CSS.Property.Inset | number, "sizes">;
    /**
     * The CSS `position` property
     */
    pos?: Token<CSS.Property.Position>;
    /**
     * The CSS `position` property
     */
    position?: Token<CSS.Property.Position>;
    insetInline?: Token<CSS.Property.InsetInline>;
    insetBlock?: Token<CSS.Property.InsetBlock>;
}
//# sourceMappingURL=position.d.ts.map