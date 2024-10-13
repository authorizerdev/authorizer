import type { SystemProps, SystemStyleObject, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
interface AvatarOptions {
    /**
     * The name of the person in the avatar.
     *
     * - if `src` has loaded, the name will be used as the `alt` attribute of the `img`
     * - If `src` is not loaded, the name will be used to create the initials
     */
    name?: string;
    /**
     * If `true`, the `Avatar` will show a border around it.
     *
     * Best for a group of avatars
     */
    showBorder?: boolean;
    /**
     * The badge at the bottom right corner of the avatar.
     */
    children?: React.ReactNode;
    /**
     * The image url of the `Avatar`
     */
    src?: string;
    /**
     * List of sources to use for different screen resolutions
     */
    srcSet?: string;
    /**
     * Defines loading strategy
     */
    loading?: "eager" | "lazy";
    /**
     * The border color of the avatar
     * @type SystemProps["borderColor"]
     */
    borderColor?: SystemProps["borderColor"];
    /**
     * Function called when image failed to load
     */
    onError?: () => void;
    /**
     * The default avatar used as fallback when `name`, and `src`
     * is not specified.
     * @type React.ReactElement
     */
    icon?: React.ReactElement;
    /**
     * Function to get the initials to display
     */
    getInitials?: (name: string) => string;
}
export interface AvatarBadgeProps extends HTMLChakraProps<"div"> {
}
/**
 * AvatarBadge used to show extra badge to the top-right
 * or bottom-right corner of an avatar.
 */
export declare const AvatarBadge: import("@chakra-ui/system").ComponentWithAs<"div", AvatarBadgeProps>;
export declare const baseStyle: SystemStyleObject;
export interface AvatarProps extends Omit<HTMLChakraProps<"span">, "onError">, AvatarOptions, ThemingProps<"Avatar"> {
    iconLabel?: string;
    /**
     * If `true`, opt out of the avatar's `fallback` logic and
     * renders the `img` at all times.
     */
    ignoreFallback?: boolean;
}
/**
 * Avatar component that renders an user avatar with
 * support for fallback avatar and name-only avatars
 */
export declare const Avatar: import("@chakra-ui/system").ComponentWithAs<"span", AvatarProps>;
export {};
//# sourceMappingURL=avatar.d.ts.map