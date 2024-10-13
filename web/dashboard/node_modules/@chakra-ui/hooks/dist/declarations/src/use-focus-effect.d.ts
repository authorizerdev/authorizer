import * as React from "react";
export declare type UseFocusEffectOptions = {
    shouldFocus: boolean;
    preventScroll?: boolean;
};
/**
 * React hook to focus an element conditionally
 *
 * @param ref the ref of the element to focus
 * @param options focus management options
 */
export declare function useFocusEffect<T extends HTMLElement>(ref: React.RefObject<T>, options: UseFocusEffectOptions): void;
//# sourceMappingURL=use-focus-effect.d.ts.map