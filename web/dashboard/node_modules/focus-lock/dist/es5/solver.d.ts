export declare const NEW_FOCUS = "NEW_FOCUS";
/**
 * Main solver for the "find next focus" question
 * @param innerNodes - used to control "return focus"
 * @param innerTabbables - used to control "autofocus"
 * @param outerNodes
 * @param activeElement
 * @param lastNode
 * @returns {number|string|undefined|*}
 */
export declare const newFocus: (innerNodes: HTMLElement[], innerTabbables: HTMLElement[], outerNodes: HTMLElement[], activeElement: HTMLElement | undefined, lastNode: HTMLElement | null) => number | undefined | typeof NEW_FOCUS;
