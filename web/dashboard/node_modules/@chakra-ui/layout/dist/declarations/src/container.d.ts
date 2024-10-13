import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
export interface ContainerProps extends HTMLChakraProps<"div">, ThemingProps<"Container"> {
    /**
     * If `true`, container will center its children
     * regardless of their width.
     */
    centerContent?: boolean;
}
/**
 * Layout component used to wrap app or website content
 *
 * It sets `margin-left` and `margin-right` to `auto`,
 * to keep its content centered.
 *
 * It also sets a default max-width of `60ch` (60 characters).
 */
export declare const Container: import("@chakra-ui/system").ComponentWithAs<"div", ContainerProps>;
//# sourceMappingURL=container.d.ts.map