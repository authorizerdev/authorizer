/// <reference types="react" />
/**
 * Determine if a character is a DOM floating point character
 * @see https://www.w3.org/TR/2012/WD-html-markup-20120329/datatypes.html#common.data.float
 */
export declare function isFloatingPointNumericCharacter(character: string): boolean;
/**
 * Determine if the event is a valid numeric keyboard event.
 * We use this so we can prevent non-number characters in the input
 */
export declare function isValidNumericKeyboardEvent(event: React.KeyboardEvent): boolean;
//# sourceMappingURL=utils.d.ts.map