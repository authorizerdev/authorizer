import { FocusableElement, isActiveElement } from "./tabbable";
export interface ExtendedFocusOptions extends FocusOptions {
    /**
     * Function that determines if the element is the active element
     */
    isActive?: typeof isActiveElement;
    /**
     * If true, the element will be focused in the next tick
     */
    nextTick?: boolean;
    /**
     * If true and element is an input element, the input's text will be selected
     */
    selectTextIfInput?: boolean;
}
export declare function focus(element: FocusableElement | null, options?: ExtendedFocusOptions): number;
//# sourceMappingURL=focus.d.ts.map