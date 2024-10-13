import { UseClickableProps } from "@chakra-ui/clickable";
import { Dict, LazyBehavior } from "@chakra-ui/utils";
import * as React from "react";
export declare const TabsDescendantsProvider: React.Provider<import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>>, useTabsDescendantsContext: () => import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>, useTabsDescendants: () => import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>, useTabsDescendant: (options?: {
    disabled?: boolean | undefined;
    id?: string | undefined;
} | undefined) => {
    descendants: import("@chakra-ui/descendant/src/use-descendant").UseDescendantsReturn;
    index: number;
    enabledIndex: number;
    register: (node: HTMLButtonElement | null) => void;
};
export interface UseTabsProps {
    /**
     * The orientation of the tab list.
     */
    orientation?: "vertical" | "horizontal";
    /**
     * If `true`, the tabs will be manually activated and
     * display its panel by pressing Space or Enter.
     *
     * If `false`, the tabs will be automatically activated
     * and their panel is displayed when they receive focus.
     */
    isManual?: boolean;
    /**
     * Callback when the index (controlled or un-controlled) changes.
     */
    onChange?: (index: number) => void;
    /**
     * The index of the selected tab (in controlled mode)
     */
    index?: number;
    /**
     * The initial index of the selected tab (in uncontrolled mode)
     */
    defaultIndex?: number;
    /**
     * The id of the tab
     */
    id?: string;
    /**
     * Performance 🚀:
     * If `true`, rendering of the tab panel's will be deferred until it is selected.
     */
    isLazy?: boolean;
    /**
     * Performance 🚀:
     * The lazy behavior of tab panels' content when not active.
     * Only works when `isLazy={true}`
     *
     * - "unmount": The content of inactive tab panels are always unmounted.
     * - "keepMounted": The content of inactive tab panels is initially unmounted,
     * but stays mounted when selected.
     *
     * @default "unmount"
     */
    lazyBehavior?: LazyBehavior;
    /**
     * The writing mode direction.
     *
     * - When in RTL, the left and right navigation is flipped
     */
    direction?: "rtl" | "ltr";
}
/**
 * Tabs hooks that provides all the states, and accessibility
 * helpers to keep all things working properly.
 *
 * Its returned object will be passed unto a Context Provider
 * so all child components can read from it.
 * There is no document link yet
 * @see Docs https://chakra-ui.com/docs/components/useTabs
 */
export declare function useTabs(props: UseTabsProps): {
    id: string;
    selectedIndex: number;
    focusedIndex: number;
    setSelectedIndex: React.Dispatch<React.SetStateAction<number>>;
    setFocusedIndex: React.Dispatch<React.SetStateAction<number>>;
    isManual: boolean | undefined;
    isLazy: boolean | undefined;
    lazyBehavior: LazyBehavior;
    orientation: "vertical" | "horizontal";
    descendants: import("@chakra-ui/descendant").DescendantsManager<HTMLButtonElement, {}>;
    direction: "ltr" | "rtl";
    htmlProps: {
        /**
         * The id of the tab
         */
        id?: string | undefined;
    };
};
export declare type UseTabsReturn = Omit<ReturnType<typeof useTabs>, "htmlProps" | "descendants">;
export declare const TabsProvider: React.Provider<UseTabsReturn>, useTabsContext: () => UseTabsReturn;
export interface UseTabListProps {
    children?: React.ReactNode;
    onKeyDown?: React.KeyboardEventHandler;
    ref?: React.Ref<any>;
}
/**
 * Tabs hook to manage multiple tab buttons,
 * and ensures only one tab is selected per time.
 *
 * @param props props object for the tablist
 */
export declare function useTabList<P extends UseTabListProps>(props: P): P & {
    role: string;
    "aria-orientation": "vertical" | "horizontal";
    onKeyDown: (event: React.KeyboardEvent<Element>) => void;
};
export declare type UseTabListReturn = ReturnType<typeof useTabList>;
export interface UseTabOptions {
    id?: string;
    isSelected?: boolean;
    panelId?: string;
    /**
     * If `true`, the `Tab` won't be toggleable
     */
    isDisabled?: boolean;
}
export interface UseTabProps extends Omit<UseClickableProps, "color">, UseTabOptions {
}
/**
 * Tabs hook to manage each tab button.
 *
 * A tab can be disabled and focusable, or both,
 * hence the use of `useClickable` to handle this scenario
 */
export declare function useTab<P extends UseTabProps>(props: P): {
    id: string;
    role: string;
    tabIndex: number;
    type: "button";
    "aria-selected": boolean;
    "aria-controls": string;
    onFocus: ((event: React.FocusEvent<HTMLElement, Element>) => void) | undefined;
    ref: (node: any) => void;
    "aria-disabled": boolean | undefined;
    disabled: boolean | undefined;
    onClick: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    onMouseDown: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseUp: React.MouseEventHandler<HTMLElement> | undefined;
    onKeyUp: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyDown: React.KeyboardEventHandler<HTMLElement> | undefined;
    onMouseOver: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseLeave: React.MouseEventHandler<HTMLElement> | undefined;
    defaultChecked?: boolean | undefined;
    defaultValue?: string | number | readonly string[] | undefined;
    suppressContentEditableWarning?: boolean | undefined;
    suppressHydrationWarning?: boolean | undefined;
    accessKey?: string | undefined;
    className?: string | undefined;
    contentEditable?: "inherit" | (boolean | "true" | "false") | undefined;
    contextMenu?: string | undefined;
    dir?: string | undefined;
    draggable?: (boolean | "true" | "false") | undefined;
    hidden?: boolean | undefined;
    lang?: string | undefined;
    placeholder?: string | undefined;
    slot?: string | undefined;
    spellCheck?: (boolean | "true" | "false") | undefined;
    style?: React.CSSProperties | undefined;
    title?: string | undefined;
    translate?: "yes" | "no" | undefined;
    radioGroup?: string | undefined;
    about?: string | undefined;
    datatype?: string | undefined;
    inlist?: any;
    prefix?: string | undefined;
    property?: string | undefined;
    resource?: string | undefined;
    typeof?: string | undefined;
    vocab?: string | undefined;
    autoCapitalize?: string | undefined;
    autoCorrect?: string | undefined;
    autoSave?: string | undefined;
    color?: string | undefined;
    itemProp?: string | undefined;
    itemScope?: boolean | undefined;
    itemType?: string | undefined;
    itemID?: string | undefined;
    itemRef?: string | undefined;
    results?: number | undefined;
    security?: string | undefined;
    unselectable?: "on" | "off" | undefined;
    inputMode?: "search" | "text" | "none" | "tel" | "url" | "email" | "numeric" | "decimal" | undefined;
    is?: string | undefined;
    'aria-activedescendant'?: string | undefined;
    'aria-atomic'?: (boolean | "true" | "false") | undefined;
    'aria-autocomplete'?: "none" | "list" | "inline" | "both" | undefined;
    'aria-busy'?: (boolean | "true" | "false") | undefined;
    'aria-checked'?: boolean | "true" | "false" | "mixed" | undefined;
    'aria-colcount'?: number | undefined;
    'aria-colindex'?: number | undefined;
    'aria-colspan'?: number | undefined;
    'aria-current'?: boolean | "time" | "true" | "false" | "page" | "step" | "location" | "date" | undefined;
    'aria-describedby'?: string | undefined;
    'aria-details'?: string | undefined;
    'aria-dropeffect'?: "link" | "none" | "copy" | "execute" | "move" | "popup" | undefined;
    'aria-errormessage'?: string | undefined;
    'aria-expanded'?: (boolean | "true" | "false") | undefined;
    'aria-flowto'?: string | undefined;
    'aria-grabbed'?: (boolean | "true" | "false") | undefined;
    'aria-haspopup'?: boolean | "dialog" | "menu" | "grid" | "true" | "false" | "listbox" | "tree" | undefined;
    'aria-hidden'?: (boolean | "true" | "false") | undefined;
    'aria-invalid'?: boolean | "true" | "false" | "grammar" | "spelling" | undefined;
    'aria-keyshortcuts'?: string | undefined;
    'aria-label'?: string | undefined;
    'aria-labelledby'?: string | undefined;
    'aria-level'?: number | undefined;
    'aria-live'?: "off" | "assertive" | "polite" | undefined;
    'aria-modal'?: (boolean | "true" | "false") | undefined;
    'aria-multiline'?: (boolean | "true" | "false") | undefined;
    'aria-multiselectable'?: (boolean | "true" | "false") | undefined;
    'aria-orientation'?: "vertical" | "horizontal" | undefined;
    'aria-owns'?: string | undefined;
    'aria-placeholder'?: string | undefined;
    'aria-posinset'?: number | undefined;
    'aria-pressed'?: boolean | "true" | "false" | "mixed" | undefined;
    'aria-readonly'?: (boolean | "true" | "false") | undefined;
    'aria-relevant'?: "text" | "all" | "additions" | "additions removals" | "additions text" | "removals" | "removals additions" | "removals text" | "text additions" | "text removals" | undefined;
    'aria-required'?: (boolean | "true" | "false") | undefined;
    'aria-roledescription'?: string | undefined;
    'aria-rowcount'?: number | undefined;
    'aria-rowindex'?: number | undefined;
    'aria-rowspan'?: number | undefined;
    'aria-setsize'?: number | undefined;
    'aria-sort'?: "none" | "ascending" | "descending" | "other" | undefined;
    'aria-valuemax'?: number | undefined;
    'aria-valuemin'?: number | undefined;
    'aria-valuenow'?: number | undefined;
    'aria-valuetext'?: string | undefined;
    children?: React.ReactNode;
    dangerouslySetInnerHTML?: {
        __html: string;
    } | undefined;
    onCopy?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCopyCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCut?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCutCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onPaste?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onPasteCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCompositionEnd?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionEndCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionStart?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionStartCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionUpdate?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionUpdateCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onFocusCapture?: React.FocusEventHandler<HTMLElement> | undefined;
    onBlur?: React.FocusEventHandler<HTMLElement> | undefined;
    onBlurCapture?: React.FocusEventHandler<HTMLElement> | undefined;
    onChange?: React.FormEventHandler<HTMLElement> | undefined;
    onChangeCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onBeforeInput?: React.FormEventHandler<HTMLElement> | undefined;
    onBeforeInputCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onInput?: React.FormEventHandler<HTMLElement> | undefined;
    onInputCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onReset?: React.FormEventHandler<HTMLElement> | undefined;
    onResetCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onSubmit?: React.FormEventHandler<HTMLElement> | undefined;
    onSubmitCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onInvalid?: React.FormEventHandler<HTMLElement> | undefined;
    onInvalidCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onLoad?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onError?: React.ReactEventHandler<HTMLElement> | undefined;
    onErrorCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onKeyDownCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyPress?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyPressCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyUpCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onAbort?: React.ReactEventHandler<HTMLElement> | undefined;
    onAbortCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlay?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayThrough?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayThroughCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onDurationChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onDurationChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEmptied?: React.ReactEventHandler<HTMLElement> | undefined;
    onEmptiedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEncrypted?: React.ReactEventHandler<HTMLElement> | undefined;
    onEncryptedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEnded?: React.ReactEventHandler<HTMLElement> | undefined;
    onEndedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedData?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedDataCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedMetadata?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedMetadataCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadStart?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadStartCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPause?: React.ReactEventHandler<HTMLElement> | undefined;
    onPauseCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlay?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlayCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlaying?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlayingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onProgress?: React.ReactEventHandler<HTMLElement> | undefined;
    onProgressCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onRateChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onRateChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeeked?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeekedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeeking?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeekingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onStalled?: React.ReactEventHandler<HTMLElement> | undefined;
    onStalledCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSuspend?: React.ReactEventHandler<HTMLElement> | undefined;
    onSuspendCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onTimeUpdate?: React.ReactEventHandler<HTMLElement> | undefined;
    onTimeUpdateCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onVolumeChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onVolumeChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onWaiting?: React.ReactEventHandler<HTMLElement> | undefined;
    onWaitingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onAuxClick?: React.MouseEventHandler<HTMLElement> | undefined;
    onAuxClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onContextMenu?: React.MouseEventHandler<HTMLElement> | undefined;
    onContextMenuCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onDoubleClick?: React.MouseEventHandler<HTMLElement> | undefined;
    onDoubleClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onDrag?: React.DragEventHandler<HTMLElement> | undefined;
    onDragCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnd?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEndCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnter?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnterCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragExit?: React.DragEventHandler<HTMLElement> | undefined;
    onDragExitCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragLeave?: React.DragEventHandler<HTMLElement> | undefined;
    onDragLeaveCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragOver?: React.DragEventHandler<HTMLElement> | undefined;
    onDragOverCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragStart?: React.DragEventHandler<HTMLElement> | undefined;
    onDragStartCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDrop?: React.DragEventHandler<HTMLElement> | undefined;
    onDropCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onMouseDownCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseEnter?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseMove?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseMoveCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOut?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOutCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOverCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseUpCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onSelect?: React.ReactEventHandler<HTMLElement> | undefined;
    onSelectCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onTouchCancel?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchCancelCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchEnd?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchEndCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchMove?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchMoveCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchStart?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchStartCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onPointerDown?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerDownCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerMove?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerMoveCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerUp?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerUpCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerCancel?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerCancelCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerEnter?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerEnterCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerLeave?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerLeaveCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOver?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOverCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOut?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOutCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onGotPointerCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onGotPointerCaptureCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onLostPointerCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onLostPointerCaptureCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onScroll?: React.UIEventHandler<HTMLElement> | undefined;
    onScrollCapture?: React.UIEventHandler<HTMLElement> | undefined;
    onWheel?: React.WheelEventHandler<HTMLElement> | undefined;
    onWheelCapture?: React.WheelEventHandler<HTMLElement> | undefined;
    onAnimationStart?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationStartCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationEnd?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationEndCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationIteration?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationIterationCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onTransitionEnd?: React.TransitionEventHandler<HTMLElement> | undefined;
    onTransitionEndCapture?: React.TransitionEventHandler<HTMLElement> | undefined;
} | {
    id: string;
    role: string;
    tabIndex: number;
    type: "button";
    "aria-selected": boolean;
    "aria-controls": string;
    onFocus: ((event: React.FocusEvent<HTMLElement, Element>) => void) | undefined;
    ref: (node: any) => void;
    "data-active": import("@chakra-ui/utils").Booleanish;
    "aria-disabled": "true" | undefined;
    onClick: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    onMouseDown: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    onMouseUp: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    onKeyUp: (event: React.KeyboardEvent<HTMLElement>) => void;
    onKeyDown: (event: React.KeyboardEvent<HTMLElement>) => void;
    onMouseOver: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    onMouseLeave: (event: React.MouseEvent<HTMLElement, MouseEvent>) => void;
    defaultChecked?: boolean | undefined;
    defaultValue?: string | number | readonly string[] | undefined;
    suppressContentEditableWarning?: boolean | undefined;
    suppressHydrationWarning?: boolean | undefined;
    accessKey?: string | undefined;
    className?: string | undefined;
    contentEditable?: "inherit" | (boolean | "true" | "false") | undefined;
    contextMenu?: string | undefined;
    dir?: string | undefined;
    draggable?: (boolean | "true" | "false") | undefined;
    hidden?: boolean | undefined;
    lang?: string | undefined;
    placeholder?: string | undefined;
    slot?: string | undefined;
    spellCheck?: (boolean | "true" | "false") | undefined;
    style?: React.CSSProperties | undefined;
    title?: string | undefined;
    translate?: "yes" | "no" | undefined;
    radioGroup?: string | undefined;
    about?: string | undefined;
    datatype?: string | undefined;
    inlist?: any;
    prefix?: string | undefined;
    property?: string | undefined;
    resource?: string | undefined;
    typeof?: string | undefined;
    vocab?: string | undefined;
    autoCapitalize?: string | undefined;
    autoCorrect?: string | undefined;
    autoSave?: string | undefined;
    color?: string | undefined;
    itemProp?: string | undefined;
    itemScope?: boolean | undefined;
    itemType?: string | undefined;
    itemID?: string | undefined;
    itemRef?: string | undefined;
    results?: number | undefined;
    security?: string | undefined;
    unselectable?: "on" | "off" | undefined;
    inputMode?: "search" | "text" | "none" | "tel" | "url" | "email" | "numeric" | "decimal" | undefined;
    is?: string | undefined;
    'aria-activedescendant'?: string | undefined;
    'aria-atomic'?: (boolean | "true" | "false") | undefined;
    'aria-autocomplete'?: "none" | "list" | "inline" | "both" | undefined;
    'aria-busy'?: (boolean | "true" | "false") | undefined;
    'aria-checked'?: boolean | "true" | "false" | "mixed" | undefined;
    'aria-colcount'?: number | undefined;
    'aria-colindex'?: number | undefined;
    'aria-colspan'?: number | undefined;
    'aria-current'?: boolean | "time" | "true" | "false" | "page" | "step" | "location" | "date" | undefined;
    'aria-describedby'?: string | undefined;
    'aria-details'?: string | undefined;
    'aria-dropeffect'?: "link" | "none" | "copy" | "execute" | "move" | "popup" | undefined;
    'aria-errormessage'?: string | undefined;
    'aria-expanded'?: (boolean | "true" | "false") | undefined;
    'aria-flowto'?: string | undefined;
    'aria-grabbed'?: (boolean | "true" | "false") | undefined;
    'aria-haspopup'?: boolean | "dialog" | "menu" | "grid" | "true" | "false" | "listbox" | "tree" | undefined;
    'aria-hidden'?: (boolean | "true" | "false") | undefined;
    'aria-invalid'?: boolean | "true" | "false" | "grammar" | "spelling" | undefined;
    'aria-keyshortcuts'?: string | undefined;
    'aria-label'?: string | undefined;
    'aria-labelledby'?: string | undefined;
    'aria-level'?: number | undefined;
    'aria-live'?: "off" | "assertive" | "polite" | undefined;
    'aria-modal'?: (boolean | "true" | "false") | undefined;
    'aria-multiline'?: (boolean | "true" | "false") | undefined;
    'aria-multiselectable'?: (boolean | "true" | "false") | undefined;
    'aria-orientation'?: "vertical" | "horizontal" | undefined;
    'aria-owns'?: string | undefined;
    'aria-placeholder'?: string | undefined;
    'aria-posinset'?: number | undefined;
    'aria-pressed'?: boolean | "true" | "false" | "mixed" | undefined;
    'aria-readonly'?: (boolean | "true" | "false") | undefined;
    'aria-relevant'?: "text" | "all" | "additions" | "additions removals" | "additions text" | "removals" | "removals additions" | "removals text" | "text additions" | "text removals" | undefined;
    'aria-required'?: (boolean | "true" | "false") | undefined;
    'aria-roledescription'?: string | undefined;
    'aria-rowcount'?: number | undefined;
    'aria-rowindex'?: number | undefined;
    'aria-rowspan'?: number | undefined;
    'aria-setsize'?: number | undefined;
    'aria-sort'?: "none" | "ascending" | "descending" | "other" | undefined;
    'aria-valuemax'?: number | undefined;
    'aria-valuemin'?: number | undefined;
    'aria-valuenow'?: number | undefined;
    'aria-valuetext'?: string | undefined;
    children?: React.ReactNode;
    dangerouslySetInnerHTML?: {
        __html: string;
    } | undefined;
    onCopy?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCopyCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCut?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCutCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onPaste?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onPasteCapture?: React.ClipboardEventHandler<HTMLElement> | undefined;
    onCompositionEnd?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionEndCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionStart?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionStartCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionUpdate?: React.CompositionEventHandler<HTMLElement> | undefined;
    onCompositionUpdateCapture?: React.CompositionEventHandler<HTMLElement> | undefined;
    onFocusCapture?: React.FocusEventHandler<HTMLElement> | undefined;
    onBlur?: React.FocusEventHandler<HTMLElement> | undefined;
    onBlurCapture?: React.FocusEventHandler<HTMLElement> | undefined;
    onChange?: React.FormEventHandler<HTMLElement> | undefined;
    onChangeCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onBeforeInput?: React.FormEventHandler<HTMLElement> | undefined;
    onBeforeInputCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onInput?: React.FormEventHandler<HTMLElement> | undefined;
    onInputCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onReset?: React.FormEventHandler<HTMLElement> | undefined;
    onResetCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onSubmit?: React.FormEventHandler<HTMLElement> | undefined;
    onSubmitCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onInvalid?: React.FormEventHandler<HTMLElement> | undefined;
    onInvalidCapture?: React.FormEventHandler<HTMLElement> | undefined;
    onLoad?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onError?: React.ReactEventHandler<HTMLElement> | undefined;
    onErrorCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onKeyDownCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyPress?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyPressCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onKeyUpCapture?: React.KeyboardEventHandler<HTMLElement> | undefined;
    onAbort?: React.ReactEventHandler<HTMLElement> | undefined;
    onAbortCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlay?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayThrough?: React.ReactEventHandler<HTMLElement> | undefined;
    onCanPlayThroughCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onDurationChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onDurationChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEmptied?: React.ReactEventHandler<HTMLElement> | undefined;
    onEmptiedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEncrypted?: React.ReactEventHandler<HTMLElement> | undefined;
    onEncryptedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onEnded?: React.ReactEventHandler<HTMLElement> | undefined;
    onEndedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedData?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedDataCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedMetadata?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadedMetadataCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadStart?: React.ReactEventHandler<HTMLElement> | undefined;
    onLoadStartCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPause?: React.ReactEventHandler<HTMLElement> | undefined;
    onPauseCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlay?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlayCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlaying?: React.ReactEventHandler<HTMLElement> | undefined;
    onPlayingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onProgress?: React.ReactEventHandler<HTMLElement> | undefined;
    onProgressCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onRateChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onRateChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeeked?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeekedCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeeking?: React.ReactEventHandler<HTMLElement> | undefined;
    onSeekingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onStalled?: React.ReactEventHandler<HTMLElement> | undefined;
    onStalledCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onSuspend?: React.ReactEventHandler<HTMLElement> | undefined;
    onSuspendCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onTimeUpdate?: React.ReactEventHandler<HTMLElement> | undefined;
    onTimeUpdateCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onVolumeChange?: React.ReactEventHandler<HTMLElement> | undefined;
    onVolumeChangeCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onWaiting?: React.ReactEventHandler<HTMLElement> | undefined;
    onWaitingCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onAuxClick?: React.MouseEventHandler<HTMLElement> | undefined;
    onAuxClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onContextMenu?: React.MouseEventHandler<HTMLElement> | undefined;
    onContextMenuCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onDoubleClick?: React.MouseEventHandler<HTMLElement> | undefined;
    onDoubleClickCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onDrag?: React.DragEventHandler<HTMLElement> | undefined;
    onDragCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnd?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEndCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnter?: React.DragEventHandler<HTMLElement> | undefined;
    onDragEnterCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragExit?: React.DragEventHandler<HTMLElement> | undefined;
    onDragExitCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragLeave?: React.DragEventHandler<HTMLElement> | undefined;
    onDragLeaveCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragOver?: React.DragEventHandler<HTMLElement> | undefined;
    onDragOverCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDragStart?: React.DragEventHandler<HTMLElement> | undefined;
    onDragStartCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onDrop?: React.DragEventHandler<HTMLElement> | undefined;
    onDropCapture?: React.DragEventHandler<HTMLElement> | undefined;
    onMouseDownCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseEnter?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseMove?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseMoveCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOut?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOutCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseOverCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onMouseUpCapture?: React.MouseEventHandler<HTMLElement> | undefined;
    onSelect?: React.ReactEventHandler<HTMLElement> | undefined;
    onSelectCapture?: React.ReactEventHandler<HTMLElement> | undefined;
    onTouchCancel?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchCancelCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchEnd?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchEndCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchMove?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchMoveCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchStart?: React.TouchEventHandler<HTMLElement> | undefined;
    onTouchStartCapture?: React.TouchEventHandler<HTMLElement> | undefined;
    onPointerDown?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerDownCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerMove?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerMoveCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerUp?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerUpCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerCancel?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerCancelCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerEnter?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerEnterCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerLeave?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerLeaveCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOver?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOverCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOut?: React.PointerEventHandler<HTMLElement> | undefined;
    onPointerOutCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onGotPointerCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onGotPointerCaptureCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onLostPointerCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onLostPointerCaptureCapture?: React.PointerEventHandler<HTMLElement> | undefined;
    onScroll?: React.UIEventHandler<HTMLElement> | undefined;
    onScrollCapture?: React.UIEventHandler<HTMLElement> | undefined;
    onWheel?: React.WheelEventHandler<HTMLElement> | undefined;
    onWheelCapture?: React.WheelEventHandler<HTMLElement> | undefined;
    onAnimationStart?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationStartCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationEnd?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationEndCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationIteration?: React.AnimationEventHandler<HTMLElement> | undefined;
    onAnimationIterationCapture?: React.AnimationEventHandler<HTMLElement> | undefined;
    onTransitionEnd?: React.TransitionEventHandler<HTMLElement> | undefined;
    onTransitionEndCapture?: React.TransitionEventHandler<HTMLElement> | undefined;
};
export interface UseTabPanelsProps {
    children?: React.ReactNode;
}
/**
 * Tabs hook for managing the visibility of multiple tab panels.
 *
 * Since only one panel can be show at a time, we use `cloneElement`
 * to inject `selected` panel to each TabPanel.
 *
 * It returns a cloned version of its children with
 * all functionality included.
 */
export declare function useTabPanels<P extends UseTabPanelsProps>(props: P): P & {
    children: React.ReactElement<any, string | React.JSXElementConstructor<any>>[];
};
/**
 * Tabs hook for managing the visible/hidden states
 * of the tab panel.
 *
 * @param props props object for the tab panel
 */
export declare function useTabPanel(props: Dict): {
    children: any;
    role: string;
    hidden: boolean;
    id: any;
    tabIndex: number;
};
/**
 * Tabs hook to show an animated indicators that
 * follows the active tab.
 *
 * The way we do it is by measuring the DOM Rect (or dimensions)
 * of the active tab, and return that as CSS style for
 * the indicator.
 */
export declare function useTabIndicator(): React.CSSProperties;
//# sourceMappingURL=use-tabs.d.ts.map