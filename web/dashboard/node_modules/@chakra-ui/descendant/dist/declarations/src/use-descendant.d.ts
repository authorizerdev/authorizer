/// <reference types="react" />
import { DescendantsManager, DescendantOptions } from "./descendant";
/**
 * @internal
 * React hook that initializes the DescendantsManager
 */
declare function useDescendants<T extends HTMLElement = HTMLElement, K = {}>(): DescendantsManager<T, K>;
export interface UseDescendantsReturn extends ReturnType<typeof useDescendants> {
}
export declare function createDescendantContext<T extends HTMLElement = HTMLElement, K = {}>(): readonly [import("react").Provider<DescendantsManager<T, K>>, () => DescendantsManager<T, K>, () => DescendantsManager<T, K>, (options?: DescendantOptions<K> | undefined) => {
    descendants: UseDescendantsReturn;
    index: number;
    enabledIndex: number;
    register: (node: T | null) => void;
}];
export {};
//# sourceMappingURL=use-descendant.d.ts.map