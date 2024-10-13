import { IconProps } from "@chakra-ui/icon";
import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface TagProps extends HTMLChakraProps<"span">, ThemingProps<"Tag"> {
}
/**
 * The tag component is used to label or categorize UI elements.
 * To style the tag globally, change the styles in `theme.components.Tag`
 * @see Docs https://chakra-ui.com/tag
 */
export declare const Tag: import("@chakra-ui/system").ComponentWithAs<"span", TagProps>;
export interface TagLabelProps extends HTMLChakraProps<"span"> {
}
export declare const TagLabel: import("@chakra-ui/system").ComponentWithAs<"span", TagLabelProps>;
export declare const TagLeftIcon: import("@chakra-ui/system").ComponentWithAs<"svg", IconProps>;
export declare const TagRightIcon: import("@chakra-ui/system").ComponentWithAs<"svg", IconProps>;
export interface TagCloseButtonProps extends Omit<HTMLChakraProps<"button">, "disabled"> {
    isDisabled?: boolean;
}
/**
 * TagCloseButton is used to close "remove" the tag
 * @see Docs https://chakra-ui.com/tag
 */
export declare const TagCloseButton: React.FC<TagCloseButtonProps>;
//# sourceMappingURL=tag.d.ts.map