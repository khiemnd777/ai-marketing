export type QualityQueueGeneration = {
  status: string;
  qualityCheck?: {
    status: string;
    findings: string[];
  } | null;
};

export function qualityNeedsAction(generation: QualityQueueGeneration) {
  return generation.status === "REVIEW_REQUIRED"
    || generation.status === "FAILED"
    || generation.qualityCheck?.status === "FAILED"
    || Boolean(generation.qualityCheck?.findings.length);
}
