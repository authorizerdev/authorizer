import { UsePopperProps } from "@chakra-ui/popper";
import { PropGetter } from "@chakra-ui/react-utils";
export interface UseTooltipProps extends Pick<UsePopperProps, "modifiers" | "gutter" | "offset" | "arrowPadding" | "direction" | "placement"> {
    /**
     * Delay (in ms) before showing the tooltip
     * @default 0ms
     */
    openDelay?: number;
    /**
     * Delay (in ms) before hiding the tooltip
     * @default 0ms
     */
    closeDelay?: number;
    /**
     * If `true`, the tooltip will hide on click
     */
    closeOnClick?: boolean;
    /**
     * If `true`, the tooltip will hide while the mouse
     * is down
     */
    closeOnMouseDown?: boolean;
    /**
     * Callback to run when the tooltip shows
     */
    onOpen?(): void;
    /**
     * Callback to run when the tooltip hides
     */
    onClose?(): void;
    /**
     * Custom `id` to use in place of `uuid`
     */
    id?: string;
    /**
     * If `true`, the tooltip will be shown (in controlled mode)
     */
    isOpen?: boolean;
    /**
     * If `true`, the tooltip will be initially shown
     */
    defaultIsOpen?: boolean;
    isDisabled?: boolean;
    arrowSize?: number;
    arrowShadowColor?: string;
}
export declare function useTooltip(props?: UseTooltipProps): {
    isOpen: boolean;
    show: () => void;
    hide: () => void;
    getTriggerProps: PropGetter<any, {}>;
    getTooltipProps: (props?: any, _ref?: any) => any;
    getTooltipPositionerProps: PropGetter<any, {}>;
    getArrowProps: import("@chakra-ui/react-utils").PropGetterV2<"div", import("@chakra-ui/popper").ArrowCSSVarProps>;
    getArrowInnerProps: import("@chakra-ui/react-utils").PropGetterV2<"div", {}>;
};
export declare type UseTooltipReturn = ReturnType<typeof useTooltip>;
//# sourceMappingURL=use-tooltip.d.ts.map