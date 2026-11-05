#!/usr/bin/env python3
"""Small, reproducible HTTP concurrency check (not a capacity benchmark)."""
import argparse
import concurrent.futures
import json
import math
import statistics
import time
import urllib.error
import urllib.request
from collections import Counter

def request(url):
    begin = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=8) as response:
            status = response.status
            response.read()
    except urllib.error.HTTPError as exc:
        status = exc.code
        exc.read()
    except (urllib.error.URLError, TimeoutError) as exc:
        return {"status": "network-error", "milliseconds": (time.perf_counter() - begin)*1000, "error": str(exc)}
    return {"status": status, "milliseconds": (time.perf_counter() - begin)*1000}

def percentile(values, percentage):
    ordered = sorted(values)
    return round(ordered[min(len(ordered)-1, math.ceil(len(ordered)*percentage)-1)], 2)

def run(args):
    url = args.base.rstrip("/") + "/"
    for attempt in range(50):
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                response.read()
            break
        except (urllib.error.URLError, TimeoutError, urllib.error.HTTPError):
            time.sleep(.25)
    else:
        raise RuntimeError("HTTP service did not become ready")

    start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as executor:
        results = list(executor.map(request, [url]*args.requests))
    seconds = time.perf_counter()-start
    codes = Counter(str(row["status"]) for row in results)
    latencies = [row["milliseconds"] for row in results]
    metrics_url = args.metrics.rstrip("/") + "/"
    with urllib.request.urlopen(metrics_url, timeout=5) as response:
        metrics = json.load(response)

    data = {
        "requests": args.requests,
        "concurrency": args.concurrency,
        "elapsed_seconds": round(seconds, 3),
        "requests_per_second": round(args.requests/seconds, 2),
        "latency_ms": {
            "p50": percentile(latencies, 0.50),
            "p95": percentile(latencies, 0.95),
            "max": round(max(latencies), 2),
            "mean": round(statistics.mean(latencies), 2),
        },
        "http_status": dict(codes),
        "metrics_snapshot": metrics,
    }
    print(json.dumps(data, indent=2))
    if codes.get("network-error"):
        raise AssertionError("HTTP network errors occurred")
    if codes.get("200", 0) == 0 or codes.get("429", 0) == 0:
        raise AssertionError("scenario must demonstrate both accepted and rate-limited traffic")
    # The readiness probe also counts as a request; no other client uses this
    # server in the isolated CI job.
    if metrics["requests_total"] < args.requests:
        raise AssertionError("metrics did not capture all load requests")
    if metrics["allowed_total"] == 0 or metrics["blocked_total"] == 0:
        raise AssertionError("metrics must contain both decisions")
    buckets = metrics["duration_buckets"]
    if sum(buckets.values()) != metrics["requests_total"]:
        raise AssertionError("latency buckets do not match requests total")
    if metrics["allowed_total"] + metrics["blocked_total"] + metrics["invalid_remote_addr_total"] != metrics["requests_total"]:
        raise AssertionError("decision counters do not match requests total")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default="http://127.0.0.1:8080")
    parser.add_argument("--metrics", default="http://127.0.0.1:9090")
    parser.add_argument("--requests", type=int, default=400)
    parser.add_argument("--concurrency", type=int, default=20)
    opts = parser.parse_args()
    if opts.requests < 2 or opts.concurrency < 1:
        parser.error("requests must be >= 2 and concurrency >= 1")
    run(opts)
