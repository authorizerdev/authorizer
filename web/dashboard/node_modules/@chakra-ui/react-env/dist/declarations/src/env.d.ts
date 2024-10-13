import React from "react";
interface Environment {
    window: Window;
    document: Document;
}
export declare function useEnvironment(): Environment;
export interface EnvironmentProviderProps {
    children: React.ReactNode;
    environment?: Environment;
}
export declare function EnvironmentProvider(props: EnvironmentProviderProps): JSX.Element;
export declare namespace EnvironmentProvider {
    var displayName: string;
}
export {};
//# sourceMappingURL=env.d.ts.map