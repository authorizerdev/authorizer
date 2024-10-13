import { DocumentNode } from 'graphql';

export interface TypedDocumentNode<Result = { [key: string]: any }, Variables = { [key: string]: any }> extends DocumentNode {
  /**
   * This type is used to ensure that the variables you pass in to the query are assignable to Variables
   * and that the Result is assignable to whatever you pass your result to. The method is never actually
   * implemented, but the type is valid because we list it as optional
   */
  __apiType?: (variables: Variables) => Result;
}

/**
 * Helper for extracting a TypeScript type for operation result from a TypedDocumentNode.
 * @example
 * const myQuery = { ... }; // TypedDocumentNode<R, V> 
 * type ResultType = ResultOf<typeof myQuery>; // Now it's R
 */
export type ResultOf<T> = T extends TypedDocumentNode<infer ResultType, infer VariablesType> ? ResultType : never;

/**
 * Helper for extracting a TypeScript type for operation variables from a TypedDocumentNode.
 * @example
 * const myQuery = { ... }; // TypedDocumentNode<R, V> 
 * type VariablesType = ResultOf<typeof myQuery>; // Now it's V
 */
export type VariablesOf<T> = T extends TypedDocumentNode<infer ResultType, infer VariablesType> ? VariablesType : never;
