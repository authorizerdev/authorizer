import { Booleanish, EventKeys } from "./types";
export declare function isElement(el: any): el is Element;
export declare function isHTMLElement(el: any): el is HTMLElement;
export declare function getOwnerWindow(node?: Element | null): typeof globalThis;
export declare function getOwnerDocument(node?: Element | null): Document;
export declare function getEventWindow(event: Event): typeof globalThis;
export declare function canUseDOM(): boolean;
export declare const isBrowser: boolean;
export declare const dataAttr: (condition: boolean | undefined) => Booleanish;
export declare const ariaAttr: (condition: boolean | undefined) => true | undefined;
export declare const cx: (...classNames: any[]) => string;
export declare function getActiveElement(node?: HTMLElement): HTMLElement;
export declare function contains(parent: HTMLElement | null, child: HTMLElement): boolean;
export declare function addDomEvent(target: EventTarget, eventName: string, handler: EventListener, options?: AddEventListenerOptions): () => void;
/**
 * Get the normalized event key across all browsers
 * @param event keyboard event
 */
export declare function normalizeEventKey(event: Pick<KeyboardEvent, "key" | "keyCode">): EventKeys;
export declare function getRelatedTarget(event: Pick<FocusEvent, "relatedTarget" | "target" | "currentTarget">): HTMLElement;
export declare function isRightClick(event: Pick<MouseEvent, "button">): boolean;
//# sourceMappingURL=dom.d.ts.map