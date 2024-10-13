import * as React from "react";
export declare type HideProps = ShowProps;
export declare const Hide: React.FC<HideProps>;
export interface ShowProps {
    breakpoint?: string;
    below?: string;
    above?: string;
    children?: React.ReactNode;
}
export declare const Show: React.FC<ShowProps>;
export interface UseQueryProps {
    breakpoint?: string;
    below?: string;
    above?: string;
}
export declare function useQuery(props: UseQueryProps): string;
//# sourceMappingURL=media-query.d.ts.map