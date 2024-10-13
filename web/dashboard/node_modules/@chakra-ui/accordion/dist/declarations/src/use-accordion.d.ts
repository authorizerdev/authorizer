import { PropGetter } from "@chakra-ui/react-utils";
import React from "react";
export declare const AccordionDescendantsProvider: React.Provider<import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>>, useAccordionDescendantsContext: () => import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>, useAccordionDescendants: () => import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>, useAccordionDescendant: (options?: {
    disabled?: boolean | undefined;
    id?: string | undefined;
} | undefined) => {
    descendants: import("@chakra-ui/descendant/src/use-descendant").UseDescendantsReturn;
    index: number;
    enabledIndex: number;
    register: (node: HTMLButtonElement | null) => void;
};
export declare type ExpandedIndex = number | number[];
export interface UseAccordionProps {
    /**
     * If `true`, multiple accordion items can be expanded at once.
     */
    allowMultiple?: boolean;
    /**
     * If `true`, any expanded accordion item can be collapsed again.
     */
    allowToggle?: boolean;
    /**
     * The index(es) of the expanded accordion item
     */
    index?: ExpandedIndex;
    /**
     * The initial index(es) of the expanded accordion item
     */
    defaultIndex?: ExpandedIndex;
    /**
     * The callback invoked when accordion items are expanded or collapsed.
     */
    onChange?(expandedIndex: ExpandedIndex): void;
}
/**
 * useAccordion hook provides all the state and focus management logic
 * for accordion items.
 */
export declare function useAccordion(props: UseAccordionProps): {
    index: ExpandedIndex;
    setIndex: React.Dispatch<React.SetStateAction<ExpandedIndex>>;
    htmlProps: {};
    getAccordionItemProps: (idx: number | null) => {
        isOpen: boolean;
        onChange: (isOpen: boolean) => void;
    };
    focusedIndex: number;
    setFocusedIndex: React.Dispatch<React.SetStateAction<number>>;
    descendants: import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>;
};
export declare type UseAccordionReturn = ReturnType<typeof useAccordion>;
interface AccordionContext extends Omit<UseAccordionReturn, "htmlProps" | "descendants"> {
    reduceMotion: boolean;
}
export declare const AccordionProvider: React.Provider<AccordionContext>, useAccordionContext: () => AccordionContext;
export interface UseAccordionItemProps {
    /**
     * If `true`, the accordion item will be disabled.
     */
    isDisabled?: boolean;
    /**
     * If `true`, the accordion item will be focusable.
     */
    isFocusable?: boolean;
    /**
     * A unique id for the accordion item.
     */
    id?: string;
}
/**
 * useAccordionItem
 *
 * React hook that provides the open/close functionality
 * for an accordion item and its children
 */
export declare function useAccordionItem(props: UseAccordionItemProps): {
    isOpen: boolean;
    isDisabled: boolean | undefined;
    isFocusable: boolean | undefined;
    onOpen: () => void;
    onClose: () => void;
    getButtonProps: PropGetter<HTMLButtonElement, {}>;
    getPanelProps: PropGetter<any, {}>;
    htmlProps: {};
};
export declare type UseAccordionItemReturn = ReturnType<typeof useAccordionItem>;
export {};
//# sourceMappingURL=use-accordion.d.ts.map