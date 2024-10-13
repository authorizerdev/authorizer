import { FocusableElement } from "@chakra-ui/utils";
import React from "react";
export interface UseFocusOnShowOptions {
    visible?: boolean;
    shouldFocus?: boolean;
    preventScroll?: boolean;
    focusRef?: React.RefObject<FocusableElement>;
}
export declare function useFocusOnShow<T extends HTMLElement>(target: React.RefObject<T> | T, options?: UseFocusOnShowOptions): void;
//# sourceMappingURL=use-focus-on-show.d.ts.map