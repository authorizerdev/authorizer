import * as React from "react";
import { IconProps } from "./icon";
interface CreateIconOptions {
    /**
     * The icon `svg` viewBox
     * @default "0 0 24 24"
     */
    viewBox?: string;
    /**
     * The `svg` path or group element
     * @type React.ReactElement | React.ReactElement[]
     */
    path?: React.ReactElement | React.ReactElement[];
    /**
     * If the has a single path, simply copy the path's `d` attribute
     */
    d?: string;
    /**
     * The display name useful in the dev tools
     */
    displayName?: string;
    /**
     * Default props automatically passed to the component; overwriteable
     */
    defaultProps?: IconProps;
}
export declare function createIcon(options: CreateIconOptions): import("@chakra-ui/system").ComponentWithAs<"svg", IconProps>;
export {};
//# sourceMappingURL=create-icon.d.ts.map