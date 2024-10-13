import { SystemProps, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
export interface TextProps extends HTMLChakraProps<"p">, ThemingProps<"Text"> {
    /**
     * The CSS `text-align` property
     * @type SystemProps["textAlign"]
     */
    align?: SystemProps["textAlign"];
    /**
     * The CSS `text-decoration` property
     * @type SystemProps["textDecoration"]
     */
    decoration?: SystemProps["textDecoration"];
    /**
     * The CSS `text-transform` property
     * @type SystemProps["textTransform"]
     */
    casing?: SystemProps["textTransform"];
}
/**
 * Used to render texts or paragraphs.
 *
 * @see Docs https://chakra-ui.com/text
 */
export declare const Text: import("@chakra-ui/system").ComponentWithAs<"p", TextProps>;
//# sourceMappingURL=text.d.ts.map