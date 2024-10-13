import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
export interface LinkProps extends HTMLChakraProps<"a">, ThemingProps<"Link"> {
    /**
     *  If `true`, the link will open in new tab
     */
    isExternal?: boolean;
}
/**
 * Links are accessible elements used primarily for navigation.
 *
 * It integrates well with other routing libraries like
 * React Router, Reach Router and Next.js Link.
 *
 * @example
 *
 * ```jsx
 * <Link as={ReactRouterLink} to="/home">Home</Link>
 * ```
 *
 * @see Docs https://chakra-ui.com/link
 */
export declare const Link: import("@chakra-ui/system").ComponentWithAs<"a", LinkProps>;
//# sourceMappingURL=link.d.ts.map