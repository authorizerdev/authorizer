export declare function getFirstItem<T>(array: T[]): T | undefined;
export declare function getLastItem<T>(array: T[]): T | undefined;
export declare function getPrevItem<T>(index: number, array: T[], loop?: boolean): T;
export declare function getNextItem<T>(index: number, array: T[], loop?: boolean): T;
export declare function removeIndex<T>(array: T[], index: number): T[];
export declare function addItem<T>(array: T[], item: T): T[];
export declare function removeItem<T>(array: T[], item: T): T[];
/**
 * Get the next index based on the current index and step.
 *
 * @param currentIndex the current index
 * @param length the total length or count of items
 * @param step the number of steps
 * @param loop whether to circle back once `currentIndex` is at the start/end
 */
export declare function getNextIndex(currentIndex: number, length: number, step?: number, loop?: boolean): number;
/**
 * Get's the previous index based on the current index.
 * Mostly used for keyboard navigation.
 *
 * @param index - the current index
 * @param count - the length or total count of items in the array
 * @param loop - whether we should circle back to the
 * first/last once `currentIndex` is at the start/end
 */
export declare function getPrevIndex(index: number, count: number, loop?: boolean): number;
/**
 * Converts an array into smaller chunks or groups.
 *
 * @param array the array to chunk into group
 * @param size the length of each chunk
 */
export declare function chunk<T>(array: T[], size: number): T[][];
/**
 * Gets the next item based on a search string
 *
 * @param items array of items
 * @param searchString the search string
 * @param itemToString resolves an item to string
 * @param currentItem the current selected item
 */
export declare function getNextItemFromSearch<T>(items: T[], searchString: string, itemToString: (item: T) => string, currentItem: T): T | undefined;
//# sourceMappingURL=array.d.ts.map