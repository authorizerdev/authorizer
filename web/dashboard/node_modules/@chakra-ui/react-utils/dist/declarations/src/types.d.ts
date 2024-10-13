import { EventKeys, Merge } from "@chakra-ui/utils";
import React, { ElementType } from "react";
export declare type MaybeRenderProp<P> = React.ReactNode | ((props: P) => React.ReactNode);
declare type WithoutStyleAttr<T> = Omit<T, "color" | "width" | "height">;
export declare type HTMLProps<T = any> = WithoutStyleAttr<React.HTMLAttributes<T>> & React.RefAttributes<T>;
export declare type PropGetter<T extends HTMLElement = any, P = {}> = (props?: Merge<HTMLProps<T>, P>, ref?: React.Ref<any> | React.RefObject<any>) => Merge<HTMLProps<T>, P>;
export declare type PropGetterV2<T extends ElementType, P = {}> = (props?: WithoutStyleAttr<React.ComponentPropsWithoutRef<T>> & P, ref?: React.Ref<any> | React.RefObject<any>) => WithoutStyleAttr<React.ComponentPropsWithRef<T>>;
export declare type EventKeyMap = Partial<Record<EventKeys, React.KeyboardEventHandler>>;
export {};
//# sourceMappingURL=types.d.ts.map