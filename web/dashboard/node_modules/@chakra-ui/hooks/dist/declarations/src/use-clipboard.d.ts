export interface UseClipboardOptions {
    /**
     * timeout delay (in ms) to switch back to initial state once copied.
     */
    timeout?: number;
    /**
     * Set the desired MIME type
     */
    format?: string;
}
/**
 * React hook to copy content to clipboard
 *
 * @param text the text or value to copy
 * @param {Number} [optionsOrTimeout=1500] optionsOrTimeout - delay (in ms) to switch back to initial state once copied.
 * @param {Object} optionsOrTimeout
 * @param {string} optionsOrTimeout.format - set the desired MIME type
 * @param {number} optionsOrTimeout.timeout - delay (in ms) to switch back to initial state once copied.
 */
export declare function useClipboard(text: string, optionsOrTimeout?: number | UseClipboardOptions): {
    value: string;
    onCopy: () => void;
    hasCopied: boolean;
};
//# sourceMappingURL=use-clipboard.d.ts.map