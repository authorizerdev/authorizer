import { HTMLChakraProps } from "@chakra-ui/system";
export interface LinkOverlayProps extends HTMLChakraProps<"a"> {
    /**
     *  If `true`, the link will open in new tab
     */
    isExternal?: boolean;
}
export declare const LinkOverlay: import("@chakra-ui/system").ComponentWithAs<"a", LinkOverlayProps>;
export interface LinkBoxProps extends HTMLChakraProps<"div"> {
}
/**
 * `LinkBox` is used to wrap content areas within a link while ensuring semantic html
 *
 * @see Docs https://chakra-ui.com/docs/navigation/link-overlay
 * @see Resources https://www.sarasoueidan.com/blog/nested-links
 */
export declare const LinkBox: import("@chakra-ui/system").ComponentWithAs<"div", LinkBoxProps>;
//# sourceMappingURL=link-box.d.ts.map