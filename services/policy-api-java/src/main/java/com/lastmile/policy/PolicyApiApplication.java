package com.lastmile.policy;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.Executors;

public final class PolicyApiApplication {
    private static final int DEFAULT_PORT = 8080;
    private static final int DEFAULT_RATE_LIMIT = 120;
    private static final long LATENCY_BUDGET_MS = 120;

    private final ObjectMapper mapper;
    private final RateLimiter rateLimiter;
    private final AuditStore auditStore;
    private final PolicyConfigStore policyConfigStore;
    private final ConfigAuditStore configAuditStore;
    private final MetricsStore metricsStore;

    private PolicyApiApplication(
        ObjectMapper mapper,
        RateLimiter rateLimiter,
        AuditStore auditStore,
        PolicyConfigStore policyConfigStore,
        ConfigAuditStore configAuditStore,
        MetricsStore metricsStore
    ) {
        this.mapper = mapper;
        this.rateLimiter = rateLimiter;
        this.auditStore = auditStore;
        this.policyConfigStore = policyConfigStore;
        this.configAuditStore = configAuditStore;
        this.metricsStore = metricsStore;
    }

    public static void main(String[] args) throws IOException {
        int port = readIntEnv("PORT", DEFAULT_PORT);
        int rateLimit = readIntEnv("RATE_LIMIT_PER_MIN", DEFAULT_RATE_LIMIT);

        ObjectMapper mapper = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, true);
        RateLimiter limiter = new RateLimiter(rateLimit, Duration.ofMinutes(1));
        AuditStore auditStore = new AuditStore();
        PolicyConfigStore policyConfigStore = new PolicyConfigStore();
        ConfigAuditStore configAuditStore = new ConfigAuditStore();
        MetricsStore metricsStore = new MetricsStore();
        PolicyApiApplication app = new PolicyApiApplication(
            mapper,
            limiter,
            auditStore,
            policyConfigStore,
            configAuditStore,
            metricsStore
        );

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/api/v1/health", app::handleHealth);
        server.createContext("/api/v1/policies/evaluate", app::handleEvaluate);
        server.createContext("/api/v1/policies/config", app::handleConfig);
        server.createContext("/api/v1/audit/policies", app::handleAudit);
        server.createContext("/api/v1/audit/policies/config", app::handleConfigAudit);
        server.createContext("/api/v1/metrics", app::handleMetrics);
        server.setExecutor(Executors.newFixedThreadPool(4));
        server.start();
    }

    private void handleHealth(HttpExchange exchange) throws IOException {
        Instant start = Instant.now();
        if (!allow(exchange)) {
            return;
        }
        writeJson(exchange, 200, new StatusResponse("ok"));
        log(exchange, new LogEntry("health check", newTraceId(), "", "", 200, LATENCY_BUDGET_MS, 0, false));
        long durationMs = Duration.between(start, Instant.now()).toMillis();
        metricsStore.record("/api/v1/health", durationMs, durationMs > LATENCY_BUDGET_MS);
    }

    private void handleEvaluate(HttpExchange exchange) throws IOException {
        Instant start = Instant.now();
        String traceId = newTraceId();

        if (!allow(exchange)) {
            return;
        }
        if (!"POST".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 405, new ErrorResponse("method not allowed"));
            log(exchange, new LogEntry("method not allowed", traceId, "", "", 405, LATENCY_BUDGET_MS, 0, false));
            return;
        }

        EvaluateRequest request;
        try {
            request = mapper.readValue(exchange.getRequestBody(), EvaluateRequest.class);
        } catch (JsonProcessingException ex) {
            writeJson(exchange, 400, new ErrorResponse("invalid json"));
            log(exchange, new LogEntry("invalid request", traceId, "", "", 400, LATENCY_BUDGET_MS, 0, false));
            return;
        }

        if (request.candidates == null || request.candidates.isEmpty()) {
            writeJson(exchange, 400, new ErrorResponse("candidates must include at least one courier"));
            log(exchange, new LogEntry("validation failed", traceId, "", "", 400, LATENCY_BUDGET_MS, 0, false));
            return;
        }
        for (Candidate candidate : request.candidates) {
            if (candidate == null || candidate.courierId == null || candidate.courierId.isBlank()) {
                writeJson(exchange, 400, new ErrorResponse("courierId is required"));
                log(exchange, new LogEntry("validation failed", traceId, "", "", 400, LATENCY_BUDGET_MS, 0, false));
                return;
            }
            if (candidate.baseScore < 0) {
                writeJson(exchange, 400, new ErrorResponse("baseScore must be >= 0"));
                log(exchange, new LogEntry("validation failed", traceId, "", "", 400, LATENCY_BUDGET_MS, 0, false));
                return;
            }
        }

        String requestId = request.requestId == null || request.requestId.isBlank()
            ? UUID.randomUUID().toString()
            : request.requestId;

        String normalizedPriority = normalizePriority(request.priority);
        double multiplier = policyConfigStore.multiplierFor(normalizedPriority);

        List<Adjustment> adjustments = new ArrayList<>();
        for (Candidate candidate : request.candidates) {
            double adjusted = candidate.baseScore * multiplier;
            adjustments.add(new Adjustment(candidate.courierId, adjusted));
        }

        EvaluateResponse response = new EvaluateResponse(requestId, traceId, multiplier, adjustments);
        writeJson(exchange, 200, response);

        long durationMs = Duration.between(start, Instant.now()).toMillis();
        String courierId = request.candidates.size() == 1
            ? request.candidates.get(0).courierId
            : "batch";
        auditStore.add(new AuditEntry(
            Instant.now(),
            requestId,
            traceId,
            normalizedPriority,
            request.candidates.size(),
            multiplier
        ));
        metricsStore.record("/api/v1/policies/evaluate", durationMs, durationMs > LATENCY_BUDGET_MS);
        log(exchange, new LogEntry(
            "policy evaluated",
            traceId,
            requestId,
            courierId,
            200,
            LATENCY_BUDGET_MS,
            durationMs,
            durationMs > LATENCY_BUDGET_MS
        ));
    }

    private void handleAudit(HttpExchange exchange) throws IOException {
        if (!allow(exchange)) {
            return;
        }
        if (!"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 405, new ErrorResponse("method not allowed"));
            return;
        }
        int limit = parseLimit(queryParam(exchange, "limit"), 50);
        writeJson(exchange, 200, new AuditResponse(auditStore.list(limit)));
    }

    private void handleConfig(HttpExchange exchange) throws IOException {
        Instant start = Instant.now();
        String traceId = newTraceId();

        if (!allow(exchange)) {
            return;
        }

        if ("GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 200, new PolicyConfigResponse(policyConfigStore.list()));
            long durationMs = Duration.between(start, Instant.now()).toMillis();
            metricsStore.record("/api/v1/policies/config", durationMs, durationMs > LATENCY_BUDGET_MS);
            log(exchange, new LogEntry(
                "policy config listed",
                traceId,
                "",
                "",
                200,
                LATENCY_BUDGET_MS,
                durationMs,
                durationMs > LATENCY_BUDGET_MS
            ));
            return;
        }

        if (!"POST".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 405, new ErrorResponse("method not allowed"));
            return;
        }

        ConfigUpsertRequest request;
        try {
            request = mapper.readValue(exchange.getRequestBody(), ConfigUpsertRequest.class);
        } catch (JsonProcessingException ex) {
            writeJson(exchange, 400, new ErrorResponse("invalid json"));
            return;
        }

        String priority = normalizePriority(request.priority);
        if (priority.isBlank()) {
            writeJson(exchange, 400, new ErrorResponse("priority is required"));
            return;
        }
        if (request.multiplier <= 0 || request.multiplier > 2) {
            writeJson(exchange, 400, new ErrorResponse("multiplier must be > 0 and <= 2"));
            return;
        }

        policyConfigStore.upsert(priority, request.multiplier);
        configAuditStore.add(new ConfigAuditEntry(Instant.now(), traceId, priority, request.multiplier, "upsert"));
        writeJson(exchange, 200, new PolicyConfigEntry(priority, request.multiplier));

        long durationMs = Duration.between(start, Instant.now()).toMillis();
        metricsStore.record("/api/v1/policies/config", durationMs, durationMs > LATENCY_BUDGET_MS);
        log(exchange, new LogEntry(
            "policy config updated",
            traceId,
            "",
            "",
            200,
            LATENCY_BUDGET_MS,
            durationMs,
            durationMs > LATENCY_BUDGET_MS
        ));
    }

    private void handleConfigAudit(HttpExchange exchange) throws IOException {
        if (!allow(exchange)) {
            return;
        }
        if (!"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 405, new ErrorResponse("method not allowed"));
            return;
        }
        int limit = parseLimit(queryParam(exchange, "limit"), 50);
        writeJson(exchange, 200, new ConfigAuditResponse(configAuditStore.list(limit)));
    }

    private void handleMetrics(HttpExchange exchange) throws IOException {
        if (!allow(exchange)) {
            return;
        }
        if (!"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            writeJson(exchange, 405, new ErrorResponse("method not allowed"));
            return;
        }
        writeJson(exchange, 200, metricsStore.snapshot());
    }

    private boolean allow(HttpExchange exchange) throws IOException {
        String key = clientIp(exchange);
        if (!rateLimiter.allow(key)) {
            writeJson(exchange, 429, new ErrorResponse("rate limit exceeded"));
            return false;
        }
        return true;
    }

    private void writeJson(HttpExchange exchange, int status, Object payload) throws IOException {
        byte[] data = mapper.writeValueAsBytes(payload);
        Headers headers = exchange.getResponseHeaders();
        headers.set("Content-Type", "application/json");
        exchange.sendResponseHeaders(status, data.length);
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(data);
        }
    }

    private void log(HttpExchange exchange, LogEntry entry) throws IOException {
        byte[] data = mapper.writeValueAsBytes(entry);
        System.out.println(new String(data));
    }

    private String clientIp(HttpExchange exchange) {
        String forwarded = exchange.getRequestHeaders().getFirst("X-Forwarded-For");
        if (forwarded != null && !forwarded.isBlank()) {
            return forwarded.split(",")[0].trim();
        }
        String realIp = exchange.getRequestHeaders().getFirst("X-Real-IP");
        if (realIp != null && !realIp.isBlank()) {
            return realIp.trim();
        }
        return exchange.getRemoteAddress().getAddress().getHostAddress();
    }

    private String queryParam(HttpExchange exchange, String key) {
        String query = exchange.getRequestURI().getQuery();
        if (query == null || query.isBlank()) {
            return null;
        }
        String[] parts = query.split("&");
        for (String part : parts) {
            String[] pair = part.split("=", 2);
            if (pair.length == 2 && key.equals(pair[0])) {
                return pair[1];
            }
        }
        return null;
    }

    private int parseLimit(String raw, int fallback) {
        if (raw == null || raw.isBlank()) {
            return fallback;
        }
        try {
            int value = Integer.parseInt(raw.trim());
            return value > 0 ? value : fallback;
        } catch (NumberFormatException ex) {
            return fallback;
        }
    }

    private static String normalizePriority(String priority) {
        if (priority == null) {
            return "standard";
        }
        String normalized = priority.trim().toLowerCase();
        if (normalized.isBlank()) {
            return "standard";
        }
        return normalized;
    }

    private static String newTraceId() {
        return UUID.randomUUID().toString();
    }

    private static int readIntEnv(String key, int fallback) {
        String raw = System.getenv(key);
        if (raw == null || raw.isBlank()) {
            return fallback;
        }
        try {
            int value = Integer.parseInt(raw.trim());
            return value > 0 ? value : fallback;
        } catch (NumberFormatException ex) {
            return fallback;
        }
    }

    private static final class RateLimiter {
        private final int maxRequests;
        private final Duration window;
        private final ConcurrentHashMap<String, Bucket> buckets = new ConcurrentHashMap<>();

        private RateLimiter(int maxRequests, Duration window) {
            this.maxRequests = maxRequests;
            this.window = window;
        }

        private boolean allow(String key) {
            Bucket bucket = buckets.computeIfAbsent(key, ignored -> new Bucket());
            synchronized (bucket) {
                Instant now = Instant.now();
                if (bucket.windowStart == null || Duration.between(bucket.windowStart, now).compareTo(window) >= 0) {
                    bucket.windowStart = now;
                    bucket.count = 0;
                }
                if (bucket.count >= maxRequests) {
                    return false;
                }
                bucket.count++;
                return true;
            }
        }

        private static final class Bucket {
            private Instant windowStart;
            private int count;
        }
    }

    private record StatusResponse(String status) {}

    private record ErrorResponse(String error) {}

    private record EvaluateRequest(String requestId, String priority, List<Candidate> candidates) {
        private EvaluateRequest {
            if (candidates == null) {
                candidates = List.of();
            }
        }
    }

    private record Candidate(String courierId, double baseScore) {}

    private record Adjustment(String courierId, double adjustedScore) {}

    private record EvaluateResponse(String requestId, String traceId, double multiplier, List<Adjustment> adjustments) {}

    private record AuditEntry(
        Instant timestamp,
        String requestId,
        String traceId,
        String priority,
        int candidateCount,
        double multiplier
    ) {}

    private record AuditResponse(List<AuditEntry> entries) {}

    private record PolicyConfigEntry(String priority, double multiplier) {}

    private record PolicyConfigResponse(List<PolicyConfigEntry> entries) {}

    private record ConfigUpsertRequest(String priority, double multiplier) {}

    private record ConfigAuditEntry(
        Instant timestamp,
        String traceId,
        String priority,
        double multiplier,
        String action
    ) {}

    private record ConfigAuditResponse(List<ConfigAuditEntry> entries) {}

    private record RouteMetrics(String route, long requests, long budgetExceeded, long avgDurationMs) {}

    private record MetricsResponse(Instant generatedAt, List<RouteMetrics> routes) {}

    private record LogEntry(
        String message,
        String traceId,
        String requestId,
        String courierId,
        int status,
        long budgetMs,
        long durationMs,
        boolean budgetExceeded
    ) {}

    private static final class AuditStore {
        private final CopyOnWriteArrayList<AuditEntry> entries = new CopyOnWriteArrayList<>();

        private void add(AuditEntry entry) {
            entries.add(entry);
            if (entries.size() > 1000) {
                entries.remove(0);
            }
        }

        private List<AuditEntry> list(int limit) {
            int size = entries.size();
            if (limit <= 0 || limit > size) {
                limit = size;
            }
            List<AuditEntry> result = new ArrayList<>(limit);
            for (int i = size - 1; i >= size - limit; i--) {
                if (i < 0) {
                    break;
                }
                result.add(entries.get(i));
            }
            return result;
        }
    }

    private static final class PolicyConfigStore {
        private final ConcurrentHashMap<String, Double> multipliers = new ConcurrentHashMap<>();

        private PolicyConfigStore() {
            multipliers.put("standard", 1.0);
            multipliers.put("express", 0.9);
            multipliers.put("priority", 0.85);
        }

        private double multiplierFor(String priority) {
            return multipliers.getOrDefault(normalizePriority(priority), multipliers.get("standard"));
        }

        private void upsert(String priority, double multiplier) {
            multipliers.put(normalizePriority(priority), multiplier);
        }

        private List<PolicyConfigEntry> list() {
            List<PolicyConfigEntry> result = new ArrayList<>(multipliers.size());
            for (Map.Entry<String, Double> entry : multipliers.entrySet()) {
                result.add(new PolicyConfigEntry(entry.getKey(), entry.getValue()));
            }
            result.sort((left, right) -> left.priority().compareTo(right.priority()));
            return result;
        }
    }

    private static final class ConfigAuditStore {
        private final CopyOnWriteArrayList<ConfigAuditEntry> entries = new CopyOnWriteArrayList<>();

        private void add(ConfigAuditEntry entry) {
            entries.add(entry);
            if (entries.size() > 1000) {
                entries.remove(0);
            }
        }

        private List<ConfigAuditEntry> list(int limit) {
            int size = entries.size();
            if (limit <= 0 || limit > size) {
                limit = size;
            }
            List<ConfigAuditEntry> result = new ArrayList<>(limit);
            for (int i = size - 1; i >= size - limit; i--) {
                if (i < 0) {
                    break;
                }
                result.add(entries.get(i));
            }
            return result;
        }
    }

    private static final class MetricsStore {
        private final ConcurrentHashMap<String, RouteBucket> buckets = new ConcurrentHashMap<>();

        private void record(String route, long durationMs, boolean budgetExceeded) {
            RouteBucket bucket = buckets.computeIfAbsent(route, ignored -> new RouteBucket());
            synchronized (bucket) {
                bucket.requests++;
                bucket.totalDurationMs += durationMs;
                if (budgetExceeded) {
                    bucket.budgetExceeded++;
                }
            }
        }

        private MetricsResponse snapshot() {
            List<RouteMetrics> routes = new ArrayList<>();
            for (Map.Entry<String, RouteBucket> entry : buckets.entrySet()) {
                RouteBucket bucket = entry.getValue();
                synchronized (bucket) {
                    long avg = bucket.requests == 0 ? 0 : bucket.totalDurationMs / bucket.requests;
                    routes.add(new RouteMetrics(entry.getKey(), bucket.requests, bucket.budgetExceeded, avg));
                }
            }
            routes.sort((left, right) -> left.route().compareTo(right.route()));
            return new MetricsResponse(Instant.now(), routes);
        }

        private static final class RouteBucket {
            private long requests;
            private long budgetExceeded;
            private long totalDurationMs;
        }
    }
}
