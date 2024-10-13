import { Dict } from "@chakra-ui/utils";
export default function createMediaQueries(breakpoints: Dict): MediaQuery[];
interface MediaQuery {
    breakpoint: string;
    maxWidth?: string;
    minWidth: string;
    query: string;
}
export {};
//# sourceMappingURL=create-media-query.d.ts.map