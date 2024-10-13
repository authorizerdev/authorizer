import { Config } from "../utils/prop-config";
import { ResponsiveValue, Token } from "../utils/types";
export declare const others: Config;
export interface OtherProps {
    /**
     * If `true`, hide an element visually without hiding it from screen readers.
     *
     * If `focusable`, the sr-only styles will be undone, making the element visible
     * to sighted users as well as screen readers.
     */
    srOnly?: true | "focusable";
    /**
     * The layer style object to apply.
     * Note: Styles must be located in `theme.layerStyles`
     */
    layerStyle?: Token<string & {}, "layerStyles">;
    /**
     * The text style object to apply.
     * Note: Styles must be located in `theme.textStyles`
     */
    textStyle?: Token<string & {}, "textStyles">;
    /**
     * Apply theme-aware style objects in `theme`
     *
     * @example
     * ```jsx
     * <Box apply="styles.h3">This is a div</Box>
     * ```
     *
     * This will apply styles defined in `theme.styles.h3`
     */
    apply?: ResponsiveValue<string>;
}
//# sourceMappingURL=others.d.ts.map