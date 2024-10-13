import { ColorMode } from "./color-mode.utils";
export declare const storageKey = "chakra-ui-color-mode";
declare type MaybeColorMode = ColorMode | undefined;
export interface StorageManager {
    get(init?: ColorMode): MaybeColorMode;
    set(value: ColorMode): void;
    type: "cookie" | "localStorage";
}
/**
 * Simple object to handle read-write to localStorage
 */
export declare const localStorageManager: StorageManager;
/**
 * Simple object to handle read-write to cookies
 */
export declare const cookieStorageManager: (cookies?: string) => StorageManager;
export {};
//# sourceMappingURL=storage-manager.d.ts.map