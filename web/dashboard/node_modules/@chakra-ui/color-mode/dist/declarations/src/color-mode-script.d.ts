/// <reference types="react" />
import { ConfigColorMode } from "./color-mode-provider";
export declare function setScript(initialValue: ConfigColorMode): void;
interface ColorModeScriptProps {
    initialColorMode?: ConfigColorMode;
    /**
     * Optional nonce that will be passed to the created `<script>` tag.
     */
    nonce?: string;
}
/**
 * Script to add to the root of your application when using localStorage,
 * to help prevent flash of color mode that can happen during page load.
 */
export declare const ColorModeScript: (props: ColorModeScriptProps) => JSX.Element;
export {};
//# sourceMappingURL=color-mode-script.d.ts.map