declare function getUserAgentBrowser(navigator: Navigator): "Chrome for iOS" | "Edge" | "Silk" | "Chrome" | "Firefox" | "AOSP" | "IE" | "Safari" | "WebKit" | null;
export declare type UserAgentBrowser = NonNullable<ReturnType<typeof getUserAgentBrowser>>;
declare function getUserAgentOS(navigator: Navigator): "Android" | "iOS" | "Windows" | "Mac" | "Chrome OS" | "Firefox OS" | null;
export declare type UserAgentOS = NonNullable<ReturnType<typeof getUserAgentOS>>;
export declare function detectDeviceType(navigator: Navigator): "tablet" | "phone" | "desktop";
export declare type UserAgentDeviceType = NonNullable<ReturnType<typeof detectDeviceType>>;
export declare function detectOS(os: UserAgentOS): boolean;
export declare function detectBrowser(browser: UserAgentBrowser): boolean;
export declare function detectTouch(): boolean;
export {};
//# sourceMappingURL=user-agent.d.ts.map