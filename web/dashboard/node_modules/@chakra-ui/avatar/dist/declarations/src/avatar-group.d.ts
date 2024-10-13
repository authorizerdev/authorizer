import { SystemProps, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
interface AvatarGroupOptions {
    /**
     * The children of the avatar group.
     *
     * Ideally should be `Avatar` and `MoreIndicator` components
     */
    children: React.ReactNode;
    /**
     * The space between the avatars in the group.
     * @type SystemProps["margin"]
     */
    spacing?: SystemProps["margin"];
    /**
     * The maximum number of visible avatars
     */
    max?: number;
}
export interface AvatarGroupProps extends AvatarGroupOptions, Omit<HTMLChakraProps<"div">, "children">, ThemingProps<"Avatar"> {
}
/**
 * AvatarGroup displays a number of avatars grouped together in a stack.
 */
export declare const AvatarGroup: import("@chakra-ui/system").ComponentWithAs<"div", AvatarGroupProps>;
export {};
//# sourceMappingURL=avatar-group.d.ts.map