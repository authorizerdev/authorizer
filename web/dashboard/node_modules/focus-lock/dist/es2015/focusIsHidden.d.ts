/**
 * checks if focus is hidden FROM the focus-lock
 * ie contained inside a node focus-lock shall ignore
 *
 * This is a utility function coupled with {@link FOCUS_ALLOW} constant
 *
 * @returns {boolean} focus is currently is in "allow" area
 */
export declare const focusIsHidden: (inDocument?: Document) => boolean;
