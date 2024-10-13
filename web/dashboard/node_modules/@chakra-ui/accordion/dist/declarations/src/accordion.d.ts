import { IconProps } from "@chakra-ui/icon";
import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import { Omit } from "@chakra-ui/utils";
import { MaybeRenderProp } from "@chakra-ui/react-utils";
import * as React from "react";
import { UseAccordionItemProps, UseAccordionProps } from "./use-accordion";
export interface AccordionProps extends UseAccordionProps, Omit<HTMLChakraProps<"div">, keyof UseAccordionProps>, ThemingProps<"Accordion"> {
    /**
     * If `true`, height animation and transitions will be disabled.
     */
    reduceMotion?: boolean;
}
/**
 * The wrapper that provides context and focus management
 * for all accordion items.
 *
 * It wraps all accordion items in a `div` for better grouping.
 * @see Docs https://chakra-ui.com/accordion
 */
export declare const Accordion: import("@chakra-ui/system").ComponentWithAs<"div", AccordionProps>;
export interface AccordionItemProps extends Omit<HTMLChakraProps<"div">, keyof UseAccordionItemProps>, UseAccordionItemProps {
    children?: MaybeRenderProp<{
        isExpanded: boolean;
        isDisabled: boolean;
    }>;
}
/**
 * AccordionItem is a single accordion that provides the open-close
 * behavior when the accordion button is clicked.
 *
 * It also provides context for the accordion button and panel.
 */
export declare const AccordionItem: import("@chakra-ui/system").ComponentWithAs<"div", AccordionItemProps>;
/**
 * React hook to get the state and actions of an accordion item
 */
export declare function useAccordionItemState(): {
    isOpen: boolean;
    onClose: () => void;
    isDisabled: boolean | undefined;
    onOpen: () => void;
};
export interface AccordionButtonProps extends HTMLChakraProps<"button"> {
}
/**
 * AccordionButton is used expands and collapses an accordion item.
 * It must be a child of `AccordionItem`.
 *
 * Note 🚨: Each accordion button must be wrapped in an heading tag,
 * that is appropriate for the information architecture of the page.
 */
export declare const AccordionButton: import("@chakra-ui/system").ComponentWithAs<"button", AccordionButtonProps>;
export interface AccordionPanelProps extends HTMLChakraProps<"div"> {
}
/**
 * Accordion panel that holds the content for each accordion.
 * It shows and hides based on the state login from the `AccordionItem`.
 *
 * It uses the `Collapse` component to animate its height.
 */
export declare const AccordionPanel: import("@chakra-ui/system").ComponentWithAs<"div", AccordionPanelProps>;
/**
 * AccordionIcon that gives a visual cue of the open/close state of the accordion item.
 * It rotates `180deg` based on the open/close state.
 */
export declare const AccordionIcon: React.FC<IconProps>;
//# sourceMappingURL=accordion.d.ts.map