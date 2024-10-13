import { useEffect } from "react";
/**
 * Sort an array of DOM nodes according to the HTML tree order
 * @see http://www.w3.org/TR/html5/infrastructure.html#tree-order
 */
export declare function sortNodes(nodes: Node[]): Node[];
export declare const isElement: (el: any) => el is HTMLElement;
export declare function getNextIndex(current: number, max: number, loop: boolean): number;
export declare function getPrevIndex(current: number, max: number, loop: boolean): number;
export declare const useSafeLayoutEffect: typeof useEffect;
export declare const cast: <T>(value: any) => T;
//# sourceMappingURL=utils.d.ts.map