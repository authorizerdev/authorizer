import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
import { UseTabListProps, UseTabOptions, UseTabsProps } from "./use-tabs";
interface TabsOptions {
    /**
     * If `true`, tabs will stretch to width of the tablist.
     */
    isFitted?: boolean;
    /**
     * The alignment of the tabs
     */
    align?: "start" | "end" | "center";
}
export interface TabsProps extends UseTabsProps, ThemingProps<"Tabs">, Omit<HTMLChakraProps<"div">, "onChange">, TabsOptions {
    children: React.ReactNode;
}
/**
 * Tabs
 *
 * Provides context and logic for all tabs components.
 */
export declare const Tabs: import("@chakra-ui/system").ComponentWithAs<"div", TabsProps>;
export interface TabProps extends UseTabOptions, HTMLChakraProps<"button"> {
}
/**
 * Tab button used to activate a specific tab panel. It renders a `button`,
 * and is responsible for automatic and manual selection modes.
 */
export declare const Tab: import("@chakra-ui/system").ComponentWithAs<"button", TabProps>;
export interface TabListProps extends UseTabListProps, Omit<HTMLChakraProps<"div">, "onKeyDown" | "ref"> {
}
/**
 * TabList is used to manage a list of tab buttons. It renders a `div` by default,
 * and is responsible the keyboard interaction between tabs.
 */
export declare const TabList: import("@chakra-ui/system").ComponentWithAs<"div", TabListProps>;
export interface TabPanelProps extends HTMLChakraProps<"div"> {
}
/**
 * TabPanel
 * Used to render the content for a specific tab.
 */
export declare const TabPanel: import("@chakra-ui/system").ComponentWithAs<"div", TabPanelProps>;
export interface TabPanelsProps extends HTMLChakraProps<"div"> {
}
/**
 * TabPanel
 *
 * Used to manage the rendering of multiple tab panels. It uses
 * `cloneElement` to hide/show tab panels.
 *
 * It renders a `div` by default.
 */
export declare const TabPanels: import("@chakra-ui/system").ComponentWithAs<"div", TabPanelsProps>;
export interface TabIndicatorProps extends HTMLChakraProps<"div"> {
}
/**
 * TabIndicator
 *
 * Used to render an active tab indicator that animates between
 * selected tabs.
 */
export declare const TabIndicator: import("@chakra-ui/system").ComponentWithAs<"div", TabIndicatorProps>;
export {};
//# sourceMappingURL=tabs.d.ts.map