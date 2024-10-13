declare type DocumentOrElement = Document | HTMLElement | null;
export declare type EventListenerEnv = (() => DocumentOrElement) | DocumentOrElement;
/**
 * React hook to manage browser event listeners
 *
 * @param event the event name
 * @param handler the event handler function to execute
 * @param doc the dom environment to execute against (defaults to `document`)
 * @param options the event listener options
 *
 * @internal
 */
export declare function useEventListener<K extends keyof DocumentEventMap>(event: K | (string & {}), handler: (event: DocumentEventMap[K]) => void, env?: EventListenerEnv, options?: boolean | AddEventListenerOptions): () => void;
export {};
//# sourceMappingURL=use-event-listener.d.ts.map