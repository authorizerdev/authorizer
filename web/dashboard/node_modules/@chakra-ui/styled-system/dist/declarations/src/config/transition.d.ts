import * as CSS from "csstype";
import { Config } from "../utils/prop-config";
import { Token } from "../utils";
export declare const transition: Config;
export interface TransitionProps {
    /**
     * The CSS `transition` property
     */
    transition?: Token<CSS.Property.Transition>;
    /**
     * The CSS `transition-property` property
     */
    transitionProperty?: Token<CSS.Property.TransitionProperty>;
    /**
     * The CSS `transition-timing-function` property
     */
    transitionTimingFunction?: Token<CSS.Property.TransitionTimingFunction>;
    /**
     * The CSS `transition-duration` property
     */
    transitionDuration?: Token<string>;
    /**
     * The CSS `transition-delay` property
     */
    transitionDelay?: Token<CSS.Property.TransitionDelay>;
    /**
     * The CSS `animation` property
     */
    animation?: Token<CSS.Property.Animation>;
    /**
     * The CSS `will-change` property
     */
    willChange?: Token<CSS.Property.WillChange>;
}
//# sourceMappingURL=transition.d.ts.map