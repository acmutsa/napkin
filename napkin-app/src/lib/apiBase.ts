/** Backend origin for Compile / Analyze (no trailing slash). */
export function apiBase(): string {
  const raw = import.meta.env.VITE_API_BASE;
  if (typeof raw === "string" && raw.trim()) {
    return raw.replace(/\/$/, "");
  }
  return "http://localhost:8080";
}

/** Default AWS region embedded in intent graph until the canvas owns region UI. */
export function awsRegionDefault(): string {
  const raw = import.meta.env.VITE_AWS_REGION;
  if (typeof raw === "string" && raw.trim()) {
    return raw.trim();
  }
  return "us-east-1";
}
