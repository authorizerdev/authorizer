import * as React from "react";
export interface PortalProps {
    /**
     * The `ref` to the component where the portal will be attached to.
     */
    containerRef?: React.RefObject<HTMLElement | null>;
    /**
     * The content or node you'll like to portal
     */
    children: React.ReactNode;
    /**
     * If `true`, the portal will check if it is within a parent portal
     * and append itself to the parent's portal node.
     * This provides nesting for portals.
     *
     * If `false`, the portal will always append to `document.body`
     * regardless of nesting. It is used to opt out of portal nesting.
     */
    appendToParentPortal?: boolean;
}
/**
 * Portal
 *
 * Declarative component used to render children into a DOM node
 * that exists outside the DOM hierarchy of the parent component.
 *
 * @see Docs https://chakra-ui.com/portal
 */
export declare function Portal(props: PortalProps): JSX.Element;
export declare namespace Portal {
    var defaultProps: {
        appendToParentPortal: boolean;
    };
    var className: string;
    var selector: string;
    var displayName: string;
}
//# sourceMappingURL=portal.d.ts.map