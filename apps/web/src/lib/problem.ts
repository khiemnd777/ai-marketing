export function errorMessage(error: unknown, fallback = "Hệ thống chưa thể hoàn tất thao tác.") {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

export function newIdempotencyKey() {
  return crypto.randomUUID();
}

export function apiError(error: unknown, fallback: string) {
  if (error && typeof error === "object" && "detail" in error && typeof error.detail === "string") return new Error(error.detail);
  return new Error(fallback);
}
