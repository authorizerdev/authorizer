import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
export interface TableContainerProps extends HTMLChakraProps<"div"> {
}
export declare const TableContainer: import("@chakra-ui/system").ComponentWithAs<"div", TableContainerProps>;
export interface TableProps extends HTMLChakraProps<"table">, ThemingProps<"Table"> {
}
export declare const Table: import("@chakra-ui/system").ComponentWithAs<"table", TableProps>;
export interface TableCaptionProps extends HTMLChakraProps<"caption"> {
    /**
     * The placement of the table caption. This sets the `caption-side` CSS attribute.
     * @default "bottom"
     */
    placement?: "top" | "bottom";
}
export declare const TableCaption: import("@chakra-ui/system").ComponentWithAs<"caption", TableCaptionProps>;
export interface TableHeadProps extends HTMLChakraProps<"thead"> {
}
export declare const Thead: import("@chakra-ui/system").ComponentWithAs<"thead", TableHeadProps>;
export interface TableBodyProps extends HTMLChakraProps<"tbody"> {
}
export declare const Tbody: import("@chakra-ui/system").ComponentWithAs<"tbody", TableBodyProps>;
export interface TableFooterProps extends HTMLChakraProps<"tfoot"> {
}
export declare const Tfoot: import("@chakra-ui/system").ComponentWithAs<"tfoot", TableFooterProps>;
export interface TableColumnHeaderProps extends HTMLChakraProps<"th"> {
    /**
     * Aligns the cell content to the right
     */
    isNumeric?: boolean;
}
export declare const Th: import("@chakra-ui/system").ComponentWithAs<"th", TableColumnHeaderProps>;
export interface TableRowProps extends HTMLChakraProps<"tr"> {
}
export declare const Tr: import("@chakra-ui/system").ComponentWithAs<"tr", TableRowProps>;
export interface TableCellProps extends HTMLChakraProps<"td"> {
    /**
     * Aligns the cell content to the right
     */
    isNumeric?: boolean;
}
export declare const Td: import("@chakra-ui/system").ComponentWithAs<"td", TableCellProps>;
//# sourceMappingURL=table.d.ts.map