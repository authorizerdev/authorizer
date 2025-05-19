interface FocusableNode {
    node: HTMLElement;
    /**
     * index in the tab order
     */
    index: number;
    /**
     * true, if this node belongs to a Lock
     */
    lockItem: boolean;
    /**
     * true, if this node is a focus-guard (system node)
     */
    guard: boolean;
}
/**
 * traverses all related nodes (including groups) returning a list of all nodes(outer and internal) with meta information
 * This is low-level API!
 * @returns list of focusable elements inside a given top(!) node.
 * @see {@link getFocusableNodes} providing a simpler API
 */
export declare const expandFocusableNodes: (topNode: HTMLElement | HTMLElement[]) => FocusableNode[];
export {};
