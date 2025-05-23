import * as allConstants from './constants';
import { focusInside } from './focusInside';
import { focusIsHidden } from './focusIsHidden';
import { focusSolver } from './focusSolver';
import { expandFocusableNodes } from './focusables';
import { moveFocusInside } from './moveFocusInside';
import { captureFocusRestore } from './return-focus';
import { focusNextElement, focusPrevElement, getRelativeFocusable, focusFirstElement, focusLastElement } from './sibling';
import { getFocusableNodes, getTabbableNodes } from './utils/DOMutils';
/**
 * magic symbols to control focus behavior from DOM
 * see description of every particular one
 */
declare const constants: typeof allConstants;
export { constants, focusInside, focusIsHidden, moveFocusInside, focusSolver, expandFocusableNodes, getFocusableNodes, getTabbableNodes, focusNextElement, focusPrevElement, focusFirstElement, focusLastElement, getRelativeFocusable, captureFocusRestore, };
/**
 * @deprecated - please use {@link moveFocusInside} named export
 */
declare const deprecated_default_moveFocusInside: typeof moveFocusInside;
export default deprecated_default_moveFocusInside;
