/**
 * @returns {Boolean} true, if the current focus is inside given node or nodes.
 * Supports nodes hidden inside shadowDom
 */
export declare const focusInside: (topNode: HTMLElement | HTMLElement[], activeElement?: HTMLElement | undefined) => boolean;
