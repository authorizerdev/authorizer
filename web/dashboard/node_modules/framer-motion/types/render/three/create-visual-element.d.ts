import { CreateVisualElement } from "../types";
import { Object3DNode } from "@react-three/fiber";
export declare const createRenderState: () => {};
export declare const threeVisualElement: ({ parent, props, presenceId, blockInitialAnimation, visualState, }: import("../types").VisualElementOptions<Object3DNode<any, any>, any>, options?: {}) => import("../types").VisualElement<Object3DNode<any, any>, any>;
export declare const createVisualElement: CreateVisualElement<any>;
