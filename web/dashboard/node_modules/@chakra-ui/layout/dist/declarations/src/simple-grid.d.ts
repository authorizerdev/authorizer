import { ResponsiveValue } from "@chakra-ui/system";
import { GridProps } from "./grid";
interface SimpleGridOptions {
    /**
     * The width at which child elements will break into columns. Pass a number for pixel values or a string for any other valid CSS length.
     */
    minChildWidth?: GridProps["minWidth"];
    /**
     * The number of columns
     */
    columns?: ResponsiveValue<number>;
    /**
     * The gap between the grid items
     */
    spacing?: GridProps["gridGap"];
    /**
     * The column gap between the grid items
     */
    spacingX?: GridProps["gridGap"];
    /**
     * The row gap between the grid items
     */
    spacingY?: GridProps["gridGap"];
}
export interface SimpleGridProps extends GridProps, SimpleGridOptions {
}
/**
 * SimpleGrid
 *
 * React component make that providers a simpler interface, and
 * make its easy to create responsive grid layouts.
 *
 * @see Docs https://chakra-ui.com/simplegrid
 */
export declare const SimpleGrid: import("@chakra-ui/system").ComponentWithAs<"div", SimpleGridProps>;
export {};
//# sourceMappingURL=simple-grid.d.ts.map