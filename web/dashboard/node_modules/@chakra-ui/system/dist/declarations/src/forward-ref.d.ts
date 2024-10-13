/**
 * All credit goes to Chance (Reach UI), Haz (Reakit) and (fluentui)
 * for creating the base type definitions upon which we improved on
 */
import * as React from "react";
import { As, ComponentWithAs, PropsOf, RightJoinProps } from "./system.types";
export declare function forwardRef<Props extends object, Component extends As>(component: React.ForwardRefRenderFunction<any, RightJoinProps<PropsOf<Component>, Props> & {
    as?: As;
}>): ComponentWithAs<Component, Props>;
//# sourceMappingURL=forward-ref.d.ts.map