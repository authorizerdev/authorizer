import { MaybeRenderProp } from "@chakra-ui/react-utils";
import { HTMLChakraProps, SystemProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
import { UseMenuItemProps, UseMenuOptionGroupProps, UseMenuOptionOptions, UseMenuProps } from "./use-menu";
export interface MenuProps extends UseMenuProps, ThemingProps<"Menu"> {
    children: MaybeRenderProp<{
        isOpen: boolean;
        onClose: () => void;
        forceUpdate: (() => void) | undefined;
    }>;
}
/**
 * Menu provides context, state, and focus management
 * to its sub-components. It doesn't render any DOM node.
 */
export declare const Menu: React.FC<MenuProps>;
export interface MenuButtonProps extends HTMLChakraProps<"button"> {
}
/**
 * The trigger for the menu list. Must be a direct child of `Menu`.
 */
export declare const MenuButton: import("@chakra-ui/system").ComponentWithAs<"button", MenuButtonProps>;
export interface MenuListProps extends HTMLChakraProps<"div"> {
    rootProps?: HTMLChakraProps<"div">;
}
export declare const MenuList: import("@chakra-ui/system").ComponentWithAs<"div", MenuListProps>;
export interface StyledMenuItemProps extends HTMLChakraProps<"button"> {
}
interface MenuItemOptions extends Pick<UseMenuItemProps, "isDisabled" | "isFocusable" | "closeOnSelect"> {
    /**
     * The icon to render before the menu item's label.
     * @type React.ReactElement
     */
    icon?: React.ReactElement;
    /**
     * The spacing between the icon and menu item's label.
     * @type SystemProps["mr"]
     */
    iconSpacing?: SystemProps["mr"];
    /**
     * Right-aligned label text content, useful for displaying hotkeys.
     */
    command?: string;
    /**
     * The spacing between the command and menu item's label.
     * @type SystemProps["ml"]
     */
    commandSpacing?: SystemProps["ml"];
}
export interface MenuItemProps extends HTMLChakraProps<"button">, MenuItemOptions {
}
export declare const MenuItem: import("@chakra-ui/system").ComponentWithAs<"button", MenuItemProps>;
export interface MenuItemOptionProps extends UseMenuOptionOptions, Omit<MenuItemProps, keyof UseMenuOptionOptions> {
    /**
     * @type React.ReactElement
     */
    icon?: React.ReactElement;
    /**
     * @type SystemProps["mr"]
     */
    iconSpacing?: SystemProps["mr"];
}
export declare const MenuItemOption: import("@chakra-ui/system").ComponentWithAs<"button", MenuItemOptionProps>;
export interface MenuOptionGroupProps extends UseMenuOptionGroupProps, Omit<MenuGroupProps, "value" | "defaultValue" | "onChange"> {
}
export declare const MenuOptionGroup: React.FC<MenuOptionGroupProps>;
export interface MenuGroupProps extends HTMLChakraProps<"div"> {
}
export declare const MenuGroup: import("@chakra-ui/system").ComponentWithAs<"div", MenuGroupProps>;
export interface MenuCommandProps extends HTMLChakraProps<"span"> {
}
export declare const MenuCommand: import("@chakra-ui/system").ComponentWithAs<"span", MenuCommandProps>;
export declare const MenuIcon: React.FC<HTMLChakraProps<"span">>;
export interface MenuDividerProps extends HTMLChakraProps<"hr"> {
}
export declare const MenuDivider: React.FC<MenuDividerProps>;
export {};
//# sourceMappingURL=menu.d.ts.map