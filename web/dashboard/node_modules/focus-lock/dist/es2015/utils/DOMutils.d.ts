import { VisibilityCache } from './is';
import { NodeIndex } from './tabOrder';
/**
 * given list of focusable elements keeps the ones user can interact with
 * @param nodes
 * @param visibilityCache
 */
export declare const filterFocusable: (nodes: HTMLElement[], visibilityCache: VisibilityCache) => HTMLElement[];
export declare const filterAutoFocusable: (nodes: HTMLElement[], cache?: VisibilityCache) => HTMLElement[];
/**
 * !__WARNING__! Low level API.
 * @returns all tabbable nodes
 *
 * @see {@link getFocusableNodes} to get any focusable element
 *
 * @param topNodes - array of top level HTMLElements to search inside
 * @param visibilityCache - an cache to store intermediate measurements. Expected to be a fresh `new Map` on every call
 */
export declare const getTabbableNodes: (topNodes: Element[], visibilityCache: VisibilityCache, withGuards?: boolean | undefined) => NodeIndex[];
/**
 * !__WARNING__! Low level API.
 *
 * @returns anything "focusable", not only tabbable. The difference is in `tabIndex=-1`
 * (without guards, as long as they are not expected to be ever focused)
 *
 * @see {@link getTabbableNodes} to get only tabble nodes element
 *
 * @param topNodes - array of top level HTMLElements to search inside
 * @param visibilityCache - an cache to store intermediate measurements. Expected to be a fresh `new Map` on every call
 */
export declare const getFocusableNodes: (topNodes: Element[], visibilityCache: VisibilityCache) => NodeIndex[];
/**
 * return list of nodes which are expected to be auto-focused
 * @param topNode
 * @param visibilityCache
 */
export declare const parentAutofocusables: (topNode: Element, visibilityCache: VisibilityCache) => Element[];
export declare const contains: (scope: Element | ShadowRoot, element: Element) => boolean;
