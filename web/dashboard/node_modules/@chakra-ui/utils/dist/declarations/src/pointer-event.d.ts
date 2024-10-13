/**
 * Credit goes to `framer-motion` of this useful utilities.
 * License can be found here: https://github.com/framer/motion
 */
export declare type AnyPointerEvent = MouseEvent | TouchEvent | PointerEvent;
declare type PointType = "page" | "client";
export declare function isMouseEvent(event: AnyPointerEvent): event is MouseEvent;
export declare function isTouchEvent(event: AnyPointerEvent): event is TouchEvent;
export interface Point {
    x: number;
    y: number;
}
export interface PointerEventInfo {
    point: Point;
}
export declare type EventHandler = (event: AnyPointerEvent, info: PointerEventInfo) => void;
export declare type EventListenerWithPointInfo = (e: AnyPointerEvent, info: PointerEventInfo) => void;
export declare function extractEventInfo(event: AnyPointerEvent, pointType?: PointType): PointerEventInfo;
export declare function getViewportPointFromEvent(event: AnyPointerEvent): PointerEventInfo;
export declare const wrapPointerEventHandler: (handler: EventListenerWithPointInfo, shouldFilterPrimaryPointer?: boolean) => EventListener;
export declare function getPointerEventName(name: string): string;
export declare function addPointerEvent(target: EventTarget, eventName: string, handler: EventListenerWithPointInfo, options?: AddEventListenerOptions): () => void;
export declare function isMultiTouchEvent(event: AnyPointerEvent): boolean;
export {};
//# sourceMappingURL=pointer-event.d.ts.map