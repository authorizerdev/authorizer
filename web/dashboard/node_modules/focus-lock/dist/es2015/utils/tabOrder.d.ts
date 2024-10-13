export interface NodeIndex {
    node: HTMLInputElement;
    tabIndex: number;
    index: number;
}
export declare const tabSort: (a: NodeIndex, b: NodeIndex) => number;
export declare const orderByTabIndex: (nodes: HTMLInputElement[], filterNegative: boolean, keepGuards?: boolean | undefined) => NodeIndex[];
