import { Dict } from "@chakra-ui/utils";
import type { ThemeScale } from "../create-theme-vars";
import type { Transform } from "./types";
interface CreateTransformOptions {
    scale: ThemeScale;
    compose?: Transform;
    transform?: Transform;
}
export declare const tokenToCSSVar: (scale: ThemeScale, value: any) => (theme: Dict) => any;
export declare function createTransform(options: CreateTransformOptions): Transform;
export {};
//# sourceMappingURL=create-transform.d.ts.map