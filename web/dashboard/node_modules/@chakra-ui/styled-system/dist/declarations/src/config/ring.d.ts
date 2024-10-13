import * as CSS from "csstype";
import { Length, Token } from "../utils";
import { Config } from "../utils/prop-config";
/**
 * The parser configuration for common outline properties
 */
export declare const ring: Config;
export interface RingProps {
    /**
     * Creates outline rings with CSS `box-shadow` property
     */
    ring?: Token<Length>;
    /**
     * The color of the outline ring
     */
    ringColor?: Token<CSS.Property.Color, "colors">;
    /**
     * The thickness of the offset shadow when using outline rings
     */
    ringOffset?: Token<Length>;
    /**
     * The color of the offset shadow when adding outline rings
     */
    ringOffsetColor?: Token<CSS.Property.Color, "colors">;
    /**
     * If the outline ring should an `inset`
     */
    ringInset?: Token<"inset" | "none">;
}
//# sourceMappingURL=ring.d.ts.map