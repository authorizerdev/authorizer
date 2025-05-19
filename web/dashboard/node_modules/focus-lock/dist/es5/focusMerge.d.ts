/**
 * The main functionality of the focus-lock package
 *
 * given top node(s) and the last active element returns the element to be focused next
 * @returns element which should be focused to move focus inside
 * @param topNode
 * @param lastNode
 */
export declare const focusMerge: (topNode: Element | Element[], lastNode: Element | null) => undefined | {
    node: HTMLElement;
};
