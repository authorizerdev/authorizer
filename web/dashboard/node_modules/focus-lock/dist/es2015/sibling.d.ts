import { NodeIndex } from './utils/tabOrder';
declare type UnresolvedSolution = {};
declare type ResolvedSolution = {
    prev: NodeIndex;
    next: NodeIndex;
    first: NodeIndex;
    last: NodeIndex;
};
/**
 * for a given `element` in a given `scope` returns focusable siblings
 * @param element - base element
 * @param scope - common parent. Can be document, but better to narrow it down for performance reasons
 * @returns {prev,next} - references to a focusable element before and after
 * @returns undefined - if operation is not applicable
 */
export declare const getRelativeFocusable: (element: Element, scope: HTMLElement | HTMLElement[], useTabbables: boolean) => UnresolvedSolution | ResolvedSolution | undefined;
declare type ScopeRef = HTMLElement | HTMLElement[];
interface FocusNextOptions {
    /**
     * the component to "scope" focus in
     * @default document.body
     */
    scope?: ScopeRef;
    /**
     * enables cycling inside the scope
     * @default true
     */
    cycle?: boolean;
    /**
     * options for focus action to control it more precisely (ie. `{ preventScroll: true }`)
     */
    focusOptions?: FocusOptions;
    /**
     * scopes to only tabbable elements
     * set to false to include all focusable elements (tabindex -1)
     * @default true
     */
    onlyTabbable?: boolean;
}
/**
 * focuses next element in the tab-order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
export declare const focusNextElement: (fromElement: Element, options?: FocusNextOptions) => void;
/**
 * focuses prev element in the tab order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
export declare const focusPrevElement: (fromElement: Element, options?: FocusNextOptions) => void;
declare type FocusBoundaryOptions = Pick<FocusNextOptions, 'focusOptions' | 'onlyTabbable'>;
/**
 * focuses first element in the tab-order
 * @param {FocusNextOptions} options - focus options
 */
export declare const focusFirstElement: (scope: ScopeRef, options?: FocusBoundaryOptions) => void;
/**
 * focuses last element in the tab order
 * @param {FocusNextOptions} options - focus options
 */
export declare const focusLastElement: (scope: ScopeRef, options?: FocusBoundaryOptions) => void;
export {};
