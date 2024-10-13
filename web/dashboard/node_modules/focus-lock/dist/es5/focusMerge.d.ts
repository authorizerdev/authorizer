import { NodeIndex } from './utils/tabOrder';
export declare const getFocusMerge: (topNode: HTMLElement | HTMLElement[], lastNode: HTMLInputElement | null) => NodeIndex | {
    node: HTMLInputElement;
} | undefined;
