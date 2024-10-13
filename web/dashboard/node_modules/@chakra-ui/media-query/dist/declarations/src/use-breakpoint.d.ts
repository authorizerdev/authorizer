export interface Breakpoint {
    breakpoint: string;
    maxWidth?: string;
    minWidth: string;
}
/**
 * React hook used to get the current responsive media breakpoint.
 *
 * @param defaultBreakpoint default breakpoint name
 * (in non-window environments like SSR)
 *
 * For SSR, you can use a package like [is-mobile](https://github.com/kaimallea/isMobile)
 * to get the default breakpoint value from the user-agent
 */
export declare function useBreakpoint(defaultBreakpoint?: string): string | undefined;
//# sourceMappingURL=use-breakpoint.d.ts.map