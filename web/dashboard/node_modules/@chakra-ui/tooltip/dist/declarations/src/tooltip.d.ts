import { PortalProps } from "@chakra-ui/portal";
import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
import { UseTooltipProps } from "./use-tooltip";
export interface TooltipProps extends HTMLChakraProps<"div">, ThemingProps<"Tooltip">, UseTooltipProps {
    /**
     * The react component to use as the
     * trigger for the tooltip
     */
    children: React.ReactNode;
    /**
     * The label of the tooltip
     */
    label?: React.ReactNode;
    /**
     * The accessible, human friendly label to use for
     * screen readers.
     *
     * If passed, tooltip will show the content `label`
     * but expose only `aria-label` to assistive technologies
     */
    "aria-label"?: string;
    /**
     * If `true`, the tooltip will wrap its children
     * in a `<span/>` with `tabIndex=0`
     */
    shouldWrapChildren?: boolean;
    /**
     * If `true`, the tooltip will show an arrow tip
     */
    hasArrow?: boolean;
    /**
     * Props to be forwarded to the portal component
     */
    portalProps?: Pick<PortalProps, "appendToParentPortal" | "containerRef">;
}
/**
 * Tooltips display informative text when users hover, focus on, or tap an element.
 *
 * @see Docs     https://chakra-ui.com/components/tooltip
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices/#tooltip
 */
export declare const Tooltip: import("@chakra-ui/system").ComponentWithAs<"div", TooltipProps>;
//# sourceMappingURL=tooltip.d.ts.map