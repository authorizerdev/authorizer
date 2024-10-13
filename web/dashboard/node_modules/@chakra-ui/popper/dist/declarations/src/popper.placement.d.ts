import type { Placement } from "@popperjs/core";
declare type Logical = "start-start" | "start-end" | "end-start" | "end-end" | "start" | "end";
declare type PlacementWithLogical = Placement | Logical;
export type { Placement, PlacementWithLogical };
export declare function getPopperPlacement(placement: PlacementWithLogical, dir?: "ltr" | "rtl"): Placement;
//# sourceMappingURL=popper.placement.d.ts.map