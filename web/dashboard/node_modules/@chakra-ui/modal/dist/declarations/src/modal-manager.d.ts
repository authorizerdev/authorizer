import { Ref } from "react";
/**
 * Proper state management for nested modals.
 * Simplified, but inspired by material-ui's ModalManager class.
 */
declare class ModalManager {
    modals: any[];
    constructor();
    add(modal: any): void;
    remove(modal: any): void;
    isTopModal(modal: any): boolean;
}
export declare const manager: ModalManager;
export declare function useModalManager(ref: Ref<any>, isOpen?: boolean): void;
export {};
//# sourceMappingURL=modal-manager.d.ts.map