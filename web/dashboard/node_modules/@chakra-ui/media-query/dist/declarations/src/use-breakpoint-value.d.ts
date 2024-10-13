/**
 * React hook for getting the value for the current breakpoint from the
 * provided responsive values object.
 *
 * @param values
 * @param defaultBreakpoint default breakpoint name
 * (in non-window environments like SSR)
 *
 * For SSR, you can use a package like [is-mobile](https://github.com/kaimallea/isMobile)
 * to get the default breakpoint value from the user-agent
 *
 * @example
 * const width = useBreakpointValue({ base: '150px', md: '250px' })
 */
export declare function useBreakpointValue<T = any>(values: Record<string, T> | T[], defaultBreakpoint?: string): T | undefined;
//# sourceMappingURL=use-breakpoint-value.d.ts.map