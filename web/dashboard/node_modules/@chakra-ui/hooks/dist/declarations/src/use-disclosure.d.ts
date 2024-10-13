export interface UseDisclosureProps {
    isOpen?: boolean;
    defaultIsOpen?: boolean;
    onClose?(): void;
    onOpen?(): void;
    id?: string;
}
export declare function useDisclosure(props?: UseDisclosureProps): {
    isOpen: boolean;
    onOpen: () => void;
    onClose: () => void;
    onToggle: () => void;
    isControlled: boolean;
    getButtonProps: (props?: any) => any;
    getDisclosureProps: (props?: any) => any;
};
export declare type UseDisclosureReturn = ReturnType<typeof useDisclosure>;
//# sourceMappingURL=use-disclosure.d.ts.map