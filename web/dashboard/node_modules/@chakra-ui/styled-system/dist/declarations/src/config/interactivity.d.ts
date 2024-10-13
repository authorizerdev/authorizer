import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { Token, Length } from "../utils";
export declare const interactivity: Config;
export interface InteractivityProps {
    /**
     * The CSS `appearance` property
     */
    appearance?: Token<CSS.Property.Appearance>;
    /**
     * The CSS `user-select` property
     */
    userSelect?: Token<CSS.Property.UserSelect>;
    /**
     * The CSS `pointer-events` property
     */
    pointerEvents?: Token<CSS.Property.PointerEvents>;
    /**
     * The CSS `resize` property
     */
    resize?: Token<CSS.Property.Resize>;
    /**
     * The CSS `cursor` property
     */
    cursor?: Token<CSS.Property.Cursor>;
    /**
     * The CSS `outline` property
     */
    outline?: Token<CSS.Property.Outline<Length>>;
    /**
     * The CSS `outline-offset` property
     */
    outlineOffset?: Token<CSS.Property.OutlineOffset<Length>>;
    /**
     * The CSS `outline-color` property
     */
    outlineColor?: Token<CSS.Property.Color, "colors">;
}
//# sourceMappingURL=interactivity.d.ts.map