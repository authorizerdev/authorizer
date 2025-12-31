Object.defineProperty(exports, '__esModule', { value: true });

var graphql_web = require('@0no-co/graphql.web');
var fetchSource = require('./urql-core-chunk.js');
var wonka = require('wonka');

var collectTypes = (obj, types) => {
  if (Array.isArray(obj)) {
    for (var item of obj) collectTypes(item, types);
  } else if (typeof obj === 'object' && obj !== null) {
    for (var _key in obj) {
      if (_key === '__typename' && typeof obj[_key] === 'string') {
        types.add(obj[_key]);
      } else {
        collectTypes(obj[_key], types);
      }
    }
  }
  return types;
};

/** Finds and returns a list of `__typename` fields found in response data.
 *
 * @privateRemarks
 * This is used by `@urql/core`’s document `cacheExchange` to find typenames
 * in a given GraphQL response’s data.
 */
var collectTypenames = response => [...collectTypes(response, new Set())];

var formatNode = node => {
  if ('definitions' in node) {
    var definitions = [];
    for (var definition of node.definitions) {
      var newDefinition = formatNode(definition);
      definitions.push(newDefinition);
    }
    return {
      ...node,
      definitions
    };
  }
  if ('directives' in node && node.directives && node.directives.length) {
    var directives = [];
    var _directives = {};
    for (var directive of node.directives) {
      var name = directive.name.value;
      if (name[0] !== '_') {
        directives.push(directive);
      } else {
        name = name.slice(1);
      }
      _directives[name] = directive;
    }
    node = {
      ...node,
      directives,
      _directives
    };
  }
  if ('selectionSet' in node) {
    var selections = [];
    var hasTypename = node.kind === graphql_web.Kind.OPERATION_DEFINITION;
    if (node.selectionSet) {
      for (var selection of node.selectionSet.selections || []) {
        hasTypename = hasTypename || selection.kind === graphql_web.Kind.FIELD && selection.name.value === '__typename' && !selection.alias;
        var newSelection = formatNode(selection);
        selections.push(newSelection);
      }
      if (!hasTypename) {
        selections.push({
          kind: graphql_web.Kind.FIELD,
          name: {
            kind: graphql_web.Kind.NAME,
            value: '__typename'
          },
          _generated: true
        });
      }
      return {
        ...node,
        selectionSet: {
          ...node.selectionSet,
          selections
        }
      };
    }
  }
  return node;
};
var formattedDocs = new Map();

/** Formats a GraphQL document to add `__typename` fields and process client-side directives.
 *
 * @param node - a {@link DocumentNode}.
 * @returns a {@link FormattedDocument}
 *
 * @remarks
 * Cache {@link Exchange | Exchanges} will require typename introspection to
 * recognize types in a GraphQL response. To retrieve these typenames,
 * this function is used to add the `__typename` fields to non-root
 * selection sets of a GraphQL document.
 *
 * Additionally, this utility will process directives, filter out client-side
 * directives starting with an `_` underscore, and place a `_directives` dictionary
 * on selection nodes.
 *
 * This utility also preserves the internally computed key of the
 * document as created by {@link createRequest} to avoid any
 * formatting from being duplicated.
 *
 * @see {@link https://spec.graphql.org/October2021/#sec-Type-Name-Introspection} for more information
 * on typename introspection via the `__typename` field.
 */
var formatDocument = node => {
  var query = fetchSource.keyDocument(node);
  var result = formattedDocs.get(query.__key);
  if (!result) {
    formattedDocs.set(query.__key, result = formatNode(query));
    // Ensure that the hash of the resulting document won't suddenly change
    // we are marking __key as non-enumerable so when external exchanges use visit
    // to manipulate a document we won't restore the previous query due to the __key
    // property.
    Object.defineProperty(result, '__key', {
      value: query.__key,
      enumerable: false
    });
  }
  return result;
};

/** Used to recursively mark `__typename` fields in data as non-enumerable.
 *
 * @deprecated Not recommended over modelling inputs manually (See #3299)
 *
 * @remarks
 * This utility can be used to recursively copy GraphQl response data and hide
 * all `__typename` fields present on it.
 *
 * Hint: It’s not recommended to do this, unless it's absolutely necessary as
 * cloning and modifying all data of a response can be unnecessarily slow, when
 * a manual and more specific copy/mask is more efficient.
 *
 * @see {@link ClientOptions.maskTypename} for a description of how the `Client` uses this utility.
 */
var maskTypename = (data, isRoot) => {
  if (!data || typeof data !== 'object') {
    return data;
  } else if (Array.isArray(data)) {
    return data.map(d => maskTypename(d));
  } else if (data && typeof data === 'object' && (isRoot || '__typename' in data)) {
    var acc = {};
    for (var key in data) {
      if (key === '__typename') {
        Object.defineProperty(acc, '__typename', {
          enumerable: false,
          value: data.__typename
        });
      } else {
        acc[key] = maskTypename(data[key]);
      }
    }
    return acc;
  } else {
    return data;
  }
};

/** Patches a `toPromise` method onto the `Source` passed to it.
 * @param source$ - the Wonka {@link Source} to patch.
 * @returns The passed `source$` with a patched `toPromise` method as a {@link PromisifiedSource}.
 * @internal
 */
function withPromise(_source$) {
  var source$ = sink => _source$(sink);
  source$.toPromise = () => wonka.toPromise(wonka.take(1)(wonka.filter(result => !result.stale && !result.hasNext)(source$)));
  source$.then = (onResolve, onReject) => source$.toPromise().then(onResolve, onReject);
  source$.subscribe = onResult => wonka.subscribe(onResult)(source$);
  return source$;
}

/** Creates a {@link Operation} from the given parameters.
 *
 * @param kind - The {@link OperationType} of GraphQL operation, i.e. `query`, `mutation`, or `subscription`.
 * @param request - The {@link GraphQLRequest} or {@link Operation} used as a template for the new `Operation`.
 * @param context - The {@link OperationContext} `context` data for the `Operation`.
 * @returns A new {@link Operation}.
 *
 * @remarks
 * This method is both used to create new {@link Operation | Operations} as well as copy and modify existing
 * operations. While it’s not required to use this function to copy an `Operation`, it is recommended, in case
 * additional dynamic logic is added to them in the future.
 *
 * Hint: When an {@link Operation} is passed to the `request` argument, the `context` argument does not have to be
 * a complete {@link OperationContext} and will instead be combined with passed {@link Operation.context}.
 *
 * @example
 * An example of copying an existing `Operation` to modify its `context`:
 *
 * ```ts
 * makeOperation(
 *   operation.kind,
 *   operation,
 *   { requestPolicy: 'cache-first' },
 * );
 * ```
 */

function makeOperation(kind, request, context) {
  return {
    ...request,
    kind,
    context: request.context ? {
      ...request.context,
      ...context
    } : context || request.context
  };
}

/** Adds additional metadata to an `Operation`'s `context.meta` property while copying it.
 * @see {@link OperationDebugMeta} for more information on the {@link OperationContext.meta} property.
 */
var addMetadata = (operation, meta) => {
  return makeOperation(operation.kind, operation, {
    meta: {
      ...operation.context.meta,
      ...meta
    }
  });
};

var noop = () => {
  /* noop */
};

/* eslint-disable prefer-rest-params */

/** A GraphQL parse function, which may be called as a tagged template literal, returning a parsed {@link DocumentNode}.
 *
 * @remarks
 * The `gql` tag or function is used to parse a GraphQL query document into a {@link DocumentNode}.
 *
 * When used as a tagged template, `gql` will automatically merge fragment definitions into the resulting
 * document and deduplicate them.
 *
 * It enforces that all fragments have a unique name. When fragments with different definitions share a name,
 * it will log a warning in development.
 *
 * Hint: It’s recommended to use this `gql` function over other GraphQL parse functions, since it puts the parsed
 * results directly into `@urql/core`’s internal caches and prevents further unnecessary work.
 *
 * @example
 * ```ts
 * const AuthorFragment = gql`
 *   fragment AuthorDisplayComponent on Author {
 *     id
 *     name
 *   }
 * `;
 *
 * const BookFragment = gql`
 *   fragment ListBookComponent on Book {
 *     id
 *     title
 *     author {
 *       ...AuthorDisplayComponent
 *     }
 *   }
 *
 *   ${AuthorFragment}
 * `;
 *
 * const BookQuery = gql`
 *   query Book($id: ID!) {
 *     book(id: $id) {
 *       ...BookFragment
 *     }
 *   }
 *
 *   ${BookFragment}
 * `;
 * ```
 */

function gql(parts) {
  var fragmentNames = new Map();
  var definitions = [];
  var source = [];

  // Apply the entire tagged template body's definitions
  var body = Array.isArray(parts) ? parts[0] : parts || '';
  for (var i = 1; i < arguments.length; i++) {
    var value = arguments[i];
    if (value && value.definitions) {
      source.push(value);
    } else {
      body += value;
    }
    body += arguments[0][i];
  }
  source.unshift(fetchSource.keyDocument(body));
  for (var document of source) {
    for (var definition of document.definitions) {
      if (definition.kind === graphql_web.Kind.FRAGMENT_DEFINITION) {
        var name = definition.name.value;
        var _value = fetchSource.stringifyDocument(definition);
        // Fragments will be deduplicated according to this Map
        if (!fragmentNames.has(name)) {
          fragmentNames.set(name, _value);
          definitions.push(definition);
        } else if (process.env.NODE_ENV !== 'production' && fragmentNames.get(name) !== _value) {
          // Fragments with the same names is expected to have the same contents
          console.warn('[WARNING: Duplicate Fragment] A fragment with name `' + name + '` already exists in this document.\n' + 'While fragment names may not be unique across your source, each name must be unique per document.');
        }
      } else {
        definitions.push(definition);
      }
    }
  }
  return fetchSource.keyDocument({
    kind: graphql_web.Kind.DOCUMENT,
    definitions
  });
}

/* eslint-disable @typescript-eslint/no-use-before-define */
var shouldSkip = ({
  kind
}) => kind !== 'mutation' && kind !== 'query';

/** Adds unique typenames to query (for invalidating cache entries) */
var mapTypeNames = operation => {
  var query = formatDocument(operation.query);
  if (query !== operation.query) {
    var formattedOperation = makeOperation(operation.kind, operation);
    formattedOperation.query = query;
    return formattedOperation;
  } else {
    return operation;
  }
};

/** Default document cache exchange.
 *
 * @remarks
 * The default document cache in `urql` avoids sending the same GraphQL request
 * multiple times by caching it using the {@link Operation.key}. It will invalidate
 * query results automatically whenever it sees a mutation responses with matching
 * `__typename`s in their responses.
 *
 * The document cache will get the introspected `__typename` fields by modifying
 * your GraphQL operation documents using the {@link formatDocument} utility.
 *
 * This automatic invalidation strategy can fail if your query or mutation don’t
 * contain matching typenames, for instance, because the query contained an
 * empty list.
 * You can manually add hints for this exchange by specifying a list of
 * {@link OperationContext.additionalTypenames} for queries and mutations that
 * should invalidate one another.
 *
 * @see {@link https://urql.dev/goto/docs/basics/document-caching} for more information on this cache.
 */
var cacheExchange = ({
  forward,
  client,
  dispatchDebug
}) => {
  var resultCache = new Map();
  var operationCache = new Map();
  var isOperationCached = operation => operation.kind === 'query' && operation.context.requestPolicy !== 'network-only' && (operation.context.requestPolicy === 'cache-only' || resultCache.has(operation.key));
  return ops$ => {
    var cachedOps$ = wonka.map(operation => {
      var cachedResult = resultCache.get(operation.key);
      process.env.NODE_ENV !== 'production' ? dispatchDebug({
        operation,
        ...(cachedResult ? {
          type: 'cacheHit',
          message: 'The result was successfully retried from the cache'
        } : {
          type: 'cacheMiss',
          message: 'The result could not be retrieved from the cache'
        }),
        "source": "cacheExchange"
      }) : undefined;
      var result = cachedResult || fetchSource.makeResult(operation, {
        data: null
      });
      result = {
        ...result,
        operation: process.env.NODE_ENV !== 'production' ? addMetadata(operation, {
          cacheOutcome: cachedResult ? 'hit' : 'miss'
        }) : operation
      };
      if (operation.context.requestPolicy === 'cache-and-network') {
        result.stale = true;
        reexecuteOperation(client, operation);
      }
      return result;
    })(wonka.filter(op => !shouldSkip(op) && isOperationCached(op))(ops$));
    var forwardedOps$ = wonka.tap(response => {
      var {
        operation
      } = response;
      if (!operation) return;
      var typenames = operation.context.additionalTypenames || [];
      // NOTE: For now, we only respect `additionalTypenames` from subscriptions to
      // avoid unexpected breaking changes
      // We'd expect live queries or other update mechanisms to be more suitable rather
      // than using subscriptions as “signals” to reexecute queries. However, if they’re
      // just used as signals, it’s intuitive to hook them up using `additionalTypenames`
      if (response.operation.kind !== 'subscription') {
        typenames = collectTypenames(response.data).concat(typenames);
      }

      // Invalidates the cache given a mutation's response
      if (response.operation.kind === 'mutation' || response.operation.kind === 'subscription') {
        var pendingOperations = new Set();
        process.env.NODE_ENV !== 'production' ? dispatchDebug({
          type: 'cacheInvalidation',
          message: `The following typenames have been invalidated: ${typenames}`,
          operation,
          data: {
            typenames,
            response
          },
          "source": "cacheExchange"
        }) : undefined;
        for (var i = 0; i < typenames.length; i++) {
          var typeName = typenames[i];
          var operations = operationCache.get(typeName);
          if (!operations) operationCache.set(typeName, operations = new Set());
          for (var key of operations.values()) pendingOperations.add(key);
          operations.clear();
        }
        for (var _key of pendingOperations.values()) {
          if (resultCache.has(_key)) {
            operation = resultCache.get(_key).operation;
            resultCache.delete(_key);
            reexecuteOperation(client, operation);
          }
        }
      } else if (operation.kind === 'query' && response.data) {
        resultCache.set(operation.key, response);
        for (var _i = 0; _i < typenames.length; _i++) {
          var _typeName = typenames[_i];
          var _operations = operationCache.get(_typeName);
          if (!_operations) operationCache.set(_typeName, _operations = new Set());
          _operations.add(operation.key);
        }
      }
    })(forward(wonka.filter(op => op.kind !== 'query' || op.context.requestPolicy !== 'cache-only')(wonka.map(op => process.env.NODE_ENV !== 'production' ? addMetadata(op, {
      cacheOutcome: 'miss'
    }) : op)(wonka.merge([wonka.map(mapTypeNames)(wonka.filter(op => !shouldSkip(op) && !isOperationCached(op))(ops$)), wonka.filter(op => shouldSkip(op))(ops$)])))));
    return wonka.merge([cachedOps$, forwardedOps$]);
  };
};

/** Reexecutes an `Operation` with the `network-only` request policy.
 * @internal
 */
var reexecuteOperation = (client, operation) => {
  return client.reexecuteOperation(makeOperation(operation.kind, operation, {
    requestPolicy: 'network-only'
  }));
};

/** A serialized version of an {@link OperationResult}.
 *
 * @remarks
 * All properties are serialized separately as JSON strings, except for the
 * {@link CombinedError} to speed up JS parsing speed, even if a result doesn’t
 * end up being used.
 *
 * @internal
 */

/** A dictionary of {@link Operation.key} keys to serializable {@link SerializedResult} objects.
 *
 * @remarks
 * It’s not recommended to modify the serialized data manually, however, multiple payloads of
 * this dictionary may safely be merged and combined.
 */

/** Options for the `ssrExchange` allowing it to either operate on the server- or client-side. */

/** An `SSRExchange` either in server-side mode, serializing results, or client-side mode, deserializing and replaying results..
 *
 * @remarks
 * This same {@link Exchange} is used in your code both for the client-side and server-side as it’s “universal”
 * and can be put into either client-side or server-side mode using the {@link SSRExchangeParams.isClient} flag.
 *
 * In server-side mode, the `ssrExchange` will “record” results it sees from your API and provide them for you
 * to send to the client-side using the {@link SSRExchange.extractData} method.
 *
 * In client-side mode, the `ssrExchange` will use these serialized results, rehydrated either using
 * {@link SSRExchange.restoreData} or {@link SSRexchangeParams.initialState}, to replay results the
 * server-side has seen and sent before.
 *
 * Each serialized result will only be replayed once, as it’s assumed that your cache exchange will have the
 * results cached afterwards.
 */

/** Serialize an OperationResult to plain JSON */
var serializeResult = (result, includeExtensions) => {
  var serialized = {
    data: JSON.stringify(result.data),
    hasNext: result.hasNext
  };
  if (result.data !== undefined) {
    serialized.data = JSON.stringify(result.data);
  }
  if (includeExtensions && result.extensions !== undefined) {
    serialized.extensions = JSON.stringify(result.extensions);
  }
  if (result.error) {
    serialized.error = {
      graphQLErrors: result.error.graphQLErrors.map(error => {
        if (!error.path && !error.extensions) return error.message;
        return {
          message: error.message,
          path: error.path,
          extensions: error.extensions
        };
      })
    };
    if (result.error.networkError) {
      serialized.error.networkError = '' + result.error.networkError;
    }
  }
  return serialized;
};

/** Deserialize plain JSON to an OperationResult
 * @internal
 */
var deserializeResult = (operation, result, includeExtensions) => ({
  operation,
  data: result.data ? JSON.parse(result.data) : undefined,
  extensions: includeExtensions && result.extensions ? JSON.parse(result.extensions) : undefined,
  error: result.error ? new fetchSource.CombinedError({
    networkError: result.error.networkError ? new Error(result.error.networkError) : undefined,
    graphQLErrors: result.error.graphQLErrors
  }) : undefined,
  stale: false,
  hasNext: !!result.hasNext
});
var revalidated = new Set();

/** Creates a server-side rendering `Exchange` that either captures responses on the server-side or replays them on the client-side.
 *
 * @param params - An {@link SSRExchangeParams} configuration object.
 * @returns the created {@link SSRExchange}
 *
 * @remarks
 * When dealing with server-side rendering, we essentially have two {@link Client | Clients} making requests,
 * the server-side client, and the client-side one. The `ssrExchange` helps implementing a tiny cache on both
 * sides that:
 *
 * - captures results on the server-side which it can serialize,
 * - replays results on the client-side that it deserialized from the server-side.
 *
 * Hint: The `ssrExchange` is basically an exchange that acts like a replacement for any fetch exchange
 * temporarily. As such, you should place it after your cache exchange but in front of any fetch exchange.
 */
var ssrExchange = (params = {}) => {
  var staleWhileRevalidate = !!params.staleWhileRevalidate;
  var includeExtensions = !!params.includeExtensions;
  var data = {};

  // On the client-side, we delete results from the cache as they're resolved
  // this is delayed so that concurrent queries don't delete each other's data
  var invalidateQueue = [];
  var invalidate = result => {
    invalidateQueue.push(result.operation.key);
    if (invalidateQueue.length === 1) {
      Promise.resolve().then(() => {
        var key;
        while (key = invalidateQueue.shift()) {
          data[key] = null;
        }
      });
    }
  };

  // The SSR Exchange is a temporary cache that can populate results into data for suspense
  // On the client it can be used to retrieve these temporary results from a rehydrated cache
  var ssr = ({
    client,
    forward
  }) => ops$ => {
    // params.isClient tells us whether we're on the client-side
    // By default we assume that we're on the client if suspense-mode is disabled
    var isClient = params && typeof params.isClient === 'boolean' ? !!params.isClient : !client.suspense;
    var forwardedOps$ = forward(wonka.map(mapTypeNames)(wonka.filter(operation => operation.kind === 'teardown' || !data[operation.key] || !!data[operation.key].hasNext || operation.context.requestPolicy === 'network-only')(ops$)));

    // NOTE: Since below we might delete the cached entry after accessing
    // it once, cachedOps$ needs to be merged after forwardedOps$
    var cachedOps$ = wonka.map(op => {
      var serialized = data[op.key];
      var cachedResult = deserializeResult(op, serialized, includeExtensions);
      if (staleWhileRevalidate && !revalidated.has(op.key)) {
        cachedResult.stale = true;
        revalidated.add(op.key);
        reexecuteOperation(client, op);
      }
      var result = {
        ...cachedResult,
        operation: process.env.NODE_ENV !== 'production' ? addMetadata(op, {
          cacheOutcome: 'hit'
        }) : op
      };
      return result;
    })(wonka.filter(operation => operation.kind !== 'teardown' && !!data[operation.key] && operation.context.requestPolicy !== 'network-only')(ops$));
    if (!isClient) {
      // On the server we cache results in the cache as they're resolved
      forwardedOps$ = wonka.tap(result => {
        var {
          operation
        } = result;
        if (operation.kind !== 'mutation') {
          var serialized = serializeResult(result, includeExtensions);
          data[operation.key] = serialized;
        }
      })(forwardedOps$);
    } else {
      // On the client we delete results from the cache as they're resolved
      cachedOps$ = wonka.tap(invalidate)(cachedOps$);
    }
    return wonka.merge([forwardedOps$, cachedOps$]);
  };
  ssr.restoreData = restore => {
    for (var _key in restore) {
      // We only restore data that hasn't been previously invalidated
      if (data[_key] !== null) {
        data[_key] = restore[_key];
      }
    }
  };
  ssr.extractData = () => {
    var result = {};
    for (var _key2 in data) if (data[_key2] != null) result[_key2] = data[_key2];
    return result;
  };
  if (params && params.initialState) {
    ssr.restoreData(params.initialState);
  }
  return ssr;
};

/** An abstract observer-like interface.
 *
 * @remarks
 * Observer-like interfaces are passed to {@link ObservableLike.subscribe} to provide them
 * with callbacks for their events.
 *
 * @see {@link https://github.com/tc39/proposal-observable} for the full TC39 Observable proposal.
 */

/** An abstract observable-like interface.
 *
 * @remarks
 * Observable, or Observable-like interfaces, are often used by GraphQL transports to abstract
 * how they send {@link ExecutionResult | ExecutionResults} to consumers. These generally contain
 * a `subscribe` method accepting an {@link ObserverLike} structure.
 *
 * @see {@link https://github.com/tc39/proposal-observable} for the full TC39 Observable proposal.
 */

/** A more cross-compatible version of the {@link GraphQLRequest} structure.
 * {@link FetchBody} for more details
 */

/** A subscription forwarding function, which must accept a {@link SubscriptionOperation}.
 *
 * @param operation - A {@link SubscriptionOperation}
 * @returns An {@link ObservableLike} object issuing {@link ExecutionResult | ExecutionResults}.
 */

/** This is called to create a subscription and needs to be hooked up to a transport client. */

/** Generic subscription exchange factory used to either create an exchange handling just subscriptions or all operation kinds.
 *
 * @remarks
 * `subscriptionExchange` can be used to create an {@link Exchange} that either
 * handles just GraphQL subscription operations, or optionally all operations,
 * when the {@link SubscriptionExchangeOpts.enableAllOperations} flag is passed.
 *
 * The {@link SubscriptionExchangeOpts.forwardSubscription} function must
 * be provided and provides a generic input that's based on {@link Operation}
 * but is compatible with many libraries implementing GraphQL request or
 * subscription interfaces.
 */
var subscriptionExchange = ({
  forwardSubscription,
  enableAllOperations,
  isSubscriptionOperation
}) => ({
  client,
  forward
}) => {
  var createSubscriptionSource = operation => {
    var observableish = forwardSubscription(fetchSource.makeFetchBody(operation), operation);
    return wonka.make(observer => {
      var isComplete = false;
      var sub;
      var result;
      function nextResult(value) {
        observer.next(result = result ? fetchSource.mergeResultPatch(result, value) : fetchSource.makeResult(operation, value));
      }
      Promise.resolve().then(() => {
        if (isComplete) return;
        sub = observableish.subscribe({
          next: nextResult,
          error(error) {
            if (Array.isArray(error)) {
              // NOTE: This is an exception for transports that deliver `GraphQLError[]`, as part
              // of the observer’s error callback (may happen as part of `graphql-ws`).
              // We only check for arrays here, as this is an extremely “unexpected” case as the
              // shape of `ExecutionResult` is instead strictly defined.
              nextResult({
                errors: error
              });
            } else {
              observer.next(fetchSource.makeErrorResult(operation, error));
            }
            observer.complete();
          },
          complete() {
            if (!isComplete) {
              isComplete = true;
              if (operation.kind === 'subscription') {
                client.reexecuteOperation(makeOperation('teardown', operation, operation.context));
              }
              if (result && result.hasNext) {
                nextResult({
                  hasNext: false
                });
              }
              observer.complete();
            }
          }
        });
      });
      return () => {
        isComplete = true;
        if (sub) sub.unsubscribe();
      };
    });
  };
  var isSubscriptionOperationFn = isSubscriptionOperation || (operation => operation.kind === 'subscription' || !!enableAllOperations && (operation.kind === 'query' || operation.kind === 'mutation'));
  return ops$ => {
    var subscriptionResults$ = wonka.mergeMap(operation => {
      var {
        key
      } = operation;
      var teardown$ = wonka.filter(op => op.kind === 'teardown' && op.key === key)(ops$);
      return wonka.takeUntil(teardown$)(createSubscriptionSource(operation));
    })(wonka.filter(operation => operation.kind !== 'teardown' && isSubscriptionOperationFn(operation))(ops$));
    var forward$ = forward(wonka.filter(operation => operation.kind === 'teardown' || !isSubscriptionOperationFn(operation))(ops$));
    return wonka.merge([subscriptionResults$, forward$]);
  };
};

/** Simple log debugger exchange.
 *
 * @remarks
 * An exchange that logs incoming {@link Operation | Operations} and
 * {@link OperationResult | OperationResults} in development.
 *
 * This exchange is a no-op in production and often used in issue reporting
 * to understand certain usage patterns of `urql` without having access to
 * the original source code.
 *
 * Hint: When you report an issue you’re having with `urql`, adding
 * this as your first exchange and posting its output can speed up
 * issue triaging a lot!
 */
var debugExchange = ({
  forward
}) => {
  if (process.env.NODE_ENV === 'production') {
    return ops$ => forward(ops$);
  } else {
    return ops$ => wonka.tap(result =>
    // eslint-disable-next-line no-console
    console.log('[Exchange debug]: Completed operation: ', result))(forward(
    // eslint-disable-next-line no-console
    wonka.tap(op => console.log('[Exchange debug]: Incoming operation: ', op))(ops$)));
  }
};

/** Default deduplication exchange.
 * @deprecated
 * This exchange's functionality is now built into the {@link Client}.
 */
var dedupExchange = ({
  forward
}) => ops$ => forward(ops$);

/* eslint-disable @typescript-eslint/no-use-before-define */

/** Default GraphQL over HTTP fetch exchange.
 *
 * @remarks
 * The default fetch exchange in `urql` supports sending GraphQL over HTTP
 * requests, can optionally send GraphQL queries as GET requests, and
 * handles incremental multipart responses.
 *
 * This exchange does not handle persisted queries or multipart uploads.
 * Support for the former can be added using `@urql/exchange-persisted-fetch`
 * and the latter using `@urql/exchange-multipart-fetch`.
 *
 * Hint: The `fetchExchange` and the two other exchanges all use the built-in fetch
 * utilities in `@urql/core/internal`, which you can also use to implement
 * a customized fetch exchange.
 *
 * @see {@link makeFetchSource} for the shared utility calling the Fetch API.
 */
var fetchExchange = ({
  forward,
  dispatchDebug
}) => {
  return ops$ => {
    var fetchResults$ = wonka.mergeMap(operation => {
      var body = fetchSource.makeFetchBody(operation);
      var url = fetchSource.makeFetchURL(operation, body);
      var fetchOptions = fetchSource.makeFetchOptions(operation, body);
      process.env.NODE_ENV !== 'production' ? dispatchDebug({
        type: 'fetchRequest',
        message: 'A fetch request is being executed.',
        operation,
        data: {
          url,
          fetchOptions
        },
        "source": "fetchExchange"
      }) : undefined;
      var source = wonka.takeUntil(wonka.filter(op => op.kind === 'teardown' && op.key === operation.key)(ops$))(fetchSource.makeFetchSource(operation, url, fetchOptions));
      if (process.env.NODE_ENV !== 'production') {
        return wonka.onPush(result => {
          var error = !result.data ? result.error : undefined;
          process.env.NODE_ENV !== 'production' ? dispatchDebug({
            type: error ? 'fetchError' : 'fetchSuccess',
            message: `A ${error ? 'failed' : 'successful'} fetch response has been returned.`,
            operation,
            data: {
              url,
              fetchOptions,
              value: error || result
            },
            "source": "fetchExchange"
          }) : undefined;
        })(source);
      }
      return source;
    })(wonka.filter(operation => {
      return operation.kind !== 'teardown' && (operation.kind !== 'subscription' || !!operation.context.fetchSubscriptions);
    })(ops$));
    var forward$ = forward(wonka.filter(operation => {
      return operation.kind === 'teardown' || operation.kind === 'subscription' && !operation.context.fetchSubscriptions;
    })(ops$));
    return wonka.merge([fetchResults$, forward$]);
  };
};

/** Composes an array of Exchanges into a single one.
 *
 * @param exchanges - An array of {@link Exchange | Exchanges}.
 * @returns - A composed {@link Exchange}.
 *
 * @remarks
 * `composeExchanges` returns an {@link Exchange} that when instantiated
 * composes the array of passed `Exchange`s into one, calling them from
 * right to left, with the prior `Exchange`’s {@link ExchangeIO} function
 * as the {@link ExchangeInput.forward} input.
 *
 * This simply merges all exchanges into one and is used by the {@link Client}
 * to merge the `exchanges` option it receives.
 *
 * @throws
 * In development, if {@link ExchangeInput.forward} is called repeatedly
 * by an {@link Exchange} an error is thrown, since `forward()` must only
 * be called once per `Exchange`.
 */
var composeExchanges = exchanges => ({
  client,
  forward,
  dispatchDebug
}) => exchanges.reduceRight((forward, exchange) => {
  var forwarded = false;
  return exchange({
    client,
    forward(operations$) {
      if (process.env.NODE_ENV !== 'production') {
        if (forwarded) throw new Error('forward() must only be called once in each Exchange.');
        forwarded = true;
      }
      return wonka.share(forward(wonka.share(operations$)));
    },
    dispatchDebug(event) {
      process.env.NODE_ENV !== 'production' ? dispatchDebug({
        timestamp: Date.now(),
        source: exchange.name,
        ...event
      }) : undefined;
    }
  });
}, forward);

/** Options for the `mapExchange` allowing it to react to incoming operations, results, or errors. */

/** Creates an `Exchange` mapping over incoming operations, results, and/or errors.
 *
 * @param opts - A {@link MapExchangeOpts} configuration object, containing the callbacks the `mapExchange` will use.
 * @returns the created {@link Exchange}
 *
 * @remarks
 * The `mapExchange` may be used to react to or modify incoming {@link Operation | Operations}
 * and {@link OperationResult | OperationResults}. Optionally, it can also modify these
 * asynchronously, when a promise is returned from the callbacks.
 *
 * This is useful to, for instance, add an authentication token to a given request, when
 * the `@urql/exchange-auth` package would be overkill.
 *
 * It can also accept an `onError` callback, which can be used to react to incoming
 * {@link CombinedError | CombinedErrors} on results, and trigger side-effects.
 *
 */
var mapExchange = ({
  onOperation,
  onResult,
  onError
}) => {
  return ({
    forward
  }) => ops$ => {
    return wonka.mergeMap(result => {
      if (onError && result.error) onError(result.error, result.operation);
      var newResult = onResult && onResult(result) || result;
      return 'then' in newResult ? wonka.fromPromise(newResult) : wonka.fromValue(newResult);
    })(forward(wonka.mergeMap(operation => {
      var newOperation = onOperation && onOperation(operation) || operation;
      return 'then' in newOperation ? wonka.fromPromise(newOperation) : wonka.fromValue(newOperation);
    })(ops$)));
  };
};

/** Used by the `Client` as the last exchange to warn about unhandled operations.
 *
 * @remarks
 * In a normal setup, some operations may go unhandled when a {@link Client} isn’t set up
 * with the right exchanges.
 * For instance, a `Client` may be missing a fetch exchange, or an exchange handling subscriptions.
 * This {@link Exchange} is added by the `Client` automatically to log warnings about unhandled
 * {@link Operaiton | Operations} in development.
 */
var fallbackExchange = ({
  dispatchDebug
}) => ops$ => {
  if (process.env.NODE_ENV !== 'production') {
    ops$ = wonka.tap(operation => {
      if (operation.kind !== 'teardown' && process.env.NODE_ENV !== 'production') {
        var message = `No exchange has handled operations of kind "${operation.kind}". Check whether you've added an exchange responsible for these operations.`;
        process.env.NODE_ENV !== 'production' ? dispatchDebug({
          type: 'fallbackCatch',
          message,
          operation,
          "source": "fallbackExchange"
        }) : undefined;
        console.warn(message);
      }
    })(ops$);
  }

  // All operations that skipped through the entire exchange chain should be filtered from the output
  return wonka.filter(_x => false)(ops$);
};

/* eslint-disable @typescript-eslint/no-use-before-define */

/** Configuration options passed when creating a new {@link Client}.
 *
 * @remarks
 * The `ClientOptions` are passed when creating a new {@link Client}, and
 * are used to instantiate the pipeline of {@link Exchange | Exchanges}, configure
 * options used to initialize {@link OperationContext | OperationContexts}, or to
 * change the general behaviour of the {@link Client}.
 */

/** The `Client` is the central hub for your GraphQL operations and holds `urql`'s state.
 *
 * @remarks
 * The `Client` manages your active GraphQL operations and their state, and contains the
 * {@link Exchange} pipeline to execute your GraphQL operations.
 *
 * It contains methods that allow you to execute GraphQL operations manually, but the `Client`
 * is also interacted with by bindings (for React, Preact, Vue, Svelte, etc) to execute GraphQL
 * operations.
 *
 * While {@link Exchange | Exchanges} are ultimately responsible for the control flow of operations,
 * sending API requests, and caching, the `Client` still has the important responsibility for
 * creating operations, managing consumers of active operations, sharing results for operations,
 * and more tasks as a “central hub”.
 *
 * @see {@link https://urql.dev/goto/docs/architecture/#requests-and-operations-on-the-client} for more information
 * on what the `Client` is and does.
 */

var Client = function Client(opts) {
  if (process.env.NODE_ENV !== 'production' && !opts.url) {
    throw new Error('You are creating an urql-client without a url.');
  }
  var ids = 0;
  var replays = new Map();
  var active = new Map();
  var dispatched = new Set();
  var queue = [];
  var baseOpts = {
    url: opts.url,
    fetchSubscriptions: opts.fetchSubscriptions,
    fetchOptions: opts.fetchOptions,
    fetch: opts.fetch,
    preferGetMethod: opts.preferGetMethod,
    requestPolicy: opts.requestPolicy || 'cache-first'
  };

  // This subject forms the input of operations; executeOperation may be
  // called to dispatch a new operation on the subject
  var operations = wonka.makeSubject();
  function nextOperation(operation) {
    if (operation.kind === 'mutation' || operation.kind === 'teardown' || !dispatched.has(operation.key)) {
      if (operation.kind === 'teardown') {
        dispatched.delete(operation.key);
      } else if (operation.kind !== 'mutation') {
        dispatched.add(operation.key);
      }
      operations.next(operation);
    }
  }

  // We define a queued dispatcher on the subject, which empties the queue when it's
  // activated to allow `reexecuteOperation` to be trampoline-scheduled
  var isOperationBatchActive = false;
  function dispatchOperation(operation) {
    if (operation) nextOperation(operation);
    if (!isOperationBatchActive) {
      isOperationBatchActive = true;
      while (isOperationBatchActive && (operation = queue.shift())) nextOperation(operation);
      isOperationBatchActive = false;
    }
  }

  /** Defines how result streams are created */
  var makeResultSource = operation => {
    var result$ =
    // End the results stream when an active teardown event is sent
    wonka.takeUntil(wonka.filter(op => op.kind === 'teardown' && op.key === operation.key)(operations.source))(
    // Filter by matching key (or _instance if it’s set)
    wonka.filter(res => res.operation.kind === operation.kind && res.operation.key === operation.key && (!res.operation.context._instance || res.operation.context._instance === operation.context._instance))(results$));

    // Mask typename properties if the option for it is turned on
    if (opts.maskTypename) {
      result$ = wonka.map(res => ({
        ...res,
        data: maskTypename(res.data, true)
      }))(result$);
    }
    if (operation.kind !== 'query') {
      // Interrupt subscriptions and mutations when they have no more results
      result$ = wonka.takeWhile(result => !!result.hasNext, true)(result$);
    } else {
      result$ =
      // Add `stale: true` flag when a new operation is sent for queries
      wonka.switchMap(result => {
        var value$ = wonka.fromValue(result);
        return result.stale || result.hasNext ? value$ : wonka.merge([value$, wonka.map(() => {
          result.stale = true;
          return result;
        })(wonka.take(1)(wonka.filter(op => op.key === operation.key)(operations.source)))]);
      })(result$);
    }
    if (operation.kind !== 'mutation') {
      result$ =
      // Cleanup active states on end of source
      wonka.onEnd(() => {
        // Delete the active operation handle
        dispatched.delete(operation.key);
        replays.delete(operation.key);
        active.delete(operation.key);
        // Interrupt active queue
        isOperationBatchActive = false;
        // Delete all queued up operations of the same key on end
        for (var i = queue.length - 1; i >= 0; i--) if (queue[i].key === operation.key) queue.splice(i, 1);
        // Dispatch a teardown signal for the stopped operation
        nextOperation(makeOperation('teardown', operation, operation.context));
      })(
      // Store replay result
      wonka.onPush(result => {
        if (result.stale) {
          // If the current result has queued up an operation of the same
          // key, then `stale` refers to it
          for (var _operation of queue) {
            if (_operation.key === result.operation.key) {
              dispatched.delete(_operation.key);
              break;
            }
          }
        } else if (!result.hasNext) {
          dispatched.delete(operation.key);
        }
        replays.set(operation.key, result);
      })(result$));
    } else {
      result$ =
      // Send mutation operation on start
      wonka.onStart(() => {
        nextOperation(operation);
      })(result$);
    }
    return wonka.share(result$);
  };
  var instance = this instanceof Client ? this : Object.create(Client.prototype);
  var client = Object.assign(instance, {
    suspense: !!opts.suspense,
    operations$: operations.source,
    reexecuteOperation(operation) {
      // Reexecute operation only if any subscribers are still subscribed to the
      // operation's exchange results
      if (operation.kind === 'teardown') {
        dispatchOperation(operation);
      } else if (operation.kind === 'mutation' || active.has(operation.key)) {
        var queued = false;
        for (var i = 0; i < queue.length; i++) queued = queued || queue[i].key === operation.key;
        if (!queued) dispatched.delete(operation.key);
        queue.push(operation);
        Promise.resolve().then(dispatchOperation);
      }
    },
    createRequestOperation(kind, request, opts) {
      if (!opts) opts = {};
      var requestOperationType;
      if (process.env.NODE_ENV !== 'production' && kind !== 'teardown' && (requestOperationType = fetchSource.getOperationType(request.query)) !== kind) {
        throw new Error(`Expected operation of type "${kind}" but found "${requestOperationType}"`);
      }
      return makeOperation(kind, request, {
        _instance: kind === 'mutation' ? ids = ids + 1 | 0 : undefined,
        ...baseOpts,
        ...opts,
        requestPolicy: opts.requestPolicy || baseOpts.requestPolicy,
        suspense: opts.suspense || opts.suspense !== false && client.suspense
      });
    },
    executeRequestOperation(operation) {
      if (operation.kind === 'mutation') {
        return withPromise(makeResultSource(operation));
      }
      return withPromise(wonka.lazy(() => {
        var source = active.get(operation.key);
        if (!source) {
          active.set(operation.key, source = makeResultSource(operation));
        }
        source = wonka.onStart(() => {
          dispatchOperation(operation);
        })(source);
        var replay = replays.get(operation.key);
        if (operation.kind === 'query' && replay && (replay.stale || replay.hasNext)) {
          return wonka.switchMap(wonka.fromValue)(wonka.merge([source, wonka.filter(replay => replay === replays.get(operation.key))(wonka.fromValue(replay))]));
        } else {
          return source;
        }
      }));
    },
    executeQuery(query, opts) {
      var operation = client.createRequestOperation('query', query, opts);
      return client.executeRequestOperation(operation);
    },
    executeSubscription(query, opts) {
      var operation = client.createRequestOperation('subscription', query, opts);
      return client.executeRequestOperation(operation);
    },
    executeMutation(query, opts) {
      var operation = client.createRequestOperation('mutation', query, opts);
      return client.executeRequestOperation(operation);
    },
    readQuery(query, variables, context) {
      var result = null;
      wonka.subscribe(res => {
        result = res;
      })(client.query(query, variables, context)).unsubscribe();
      return result;
    },
    query(query, variables, context) {
      return client.executeQuery(fetchSource.createRequest(query, variables), context);
    },
    subscription(query, variables, context) {
      return client.executeSubscription(fetchSource.createRequest(query, variables), context);
    },
    mutation(query, variables, context) {
      return client.executeMutation(fetchSource.createRequest(query, variables), context);
    }
  });
  var dispatchDebug = noop;
  if (process.env.NODE_ENV !== 'production') {
    var {
      next,
      source
    } = wonka.makeSubject();
    client.subscribeToDebugTarget = onEvent => wonka.subscribe(onEvent)(source);
    dispatchDebug = next;
  }

  // All exchange are composed into a single one and are called using the constructed client
  // and the fallback exchange stream
  var composedExchange = composeExchanges(opts.exchanges);

  // All exchanges receive inputs using which they can forward operations to the next exchange
  // and receive a stream of results in return, access the client, or dispatch debugging events
  // All operations then run through the Exchange IOs in a pipeline-like fashion
  var results$ = wonka.share(composedExchange({
    client,
    dispatchDebug,
    forward: fallbackExchange({
      dispatchDebug
    })
  })(operations.source));

  // Prevent the `results$` exchange pipeline from being closed by active
  // cancellations cascading up from components
  wonka.publish(results$);
  return client;
};

/** Accepts `ClientOptions` and creates a `Client`.
 * @param opts - A {@link ClientOptions} objects with options for the `Client`.
 * @returns A {@link Client} instantiated with `opts`.
 */
var createClient = Client;

exports.CombinedError = fetchSource.CombinedError;
exports.createRequest = fetchSource.createRequest;
exports.makeErrorResult = fetchSource.makeErrorResult;
exports.makeResult = fetchSource.makeResult;
exports.mergeResultPatch = fetchSource.mergeResultPatch;
exports.stringifyDocument = fetchSource.stringifyDocument;
exports.stringifyVariables = fetchSource.stringifyVariables;
exports.Client = Client;
exports.cacheExchange = cacheExchange;
exports.composeExchanges = composeExchanges;
exports.createClient = createClient;
exports.debugExchange = debugExchange;
exports.dedupExchange = dedupExchange;
exports.errorExchange = mapExchange;
exports.fetchExchange = fetchExchange;
exports.formatDocument = formatDocument;
exports.gql = gql;
exports.makeOperation = makeOperation;
exports.mapExchange = mapExchange;
exports.maskTypename = maskTypename;
exports.ssrExchange = ssrExchange;
exports.subscriptionExchange = subscriptionExchange;
//# sourceMappingURL=urql-core.js.map
