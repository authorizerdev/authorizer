import { PanEventHandler } from "@chakra-ui/utils";
import React from "react";
export interface UsePanGestureProps {
    onPan?: PanEventHandler;
    onPanStart?: PanEventHandler;
    onPanEnd?: PanEventHandler;
    onPanSessionStart?: PanEventHandler;
    onPanSessionEnd?: PanEventHandler;
    threshold?: number;
}
export declare function usePanGesture(ref: React.RefObject<HTMLElement>, props: UsePanGestureProps): void;
//# sourceMappingURL=use-pan-gesture.d.ts.map