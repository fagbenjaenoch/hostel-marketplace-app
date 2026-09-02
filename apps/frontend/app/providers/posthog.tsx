import { PostHogProvider } from "@posthog/next";

export function PHProvider({ children }: { children: React.ReactNode }) {
  if (process.env.NODE_ENV !== "production") {
    return <>{children}</>;
  }

  return <PostHogProvider>{children}</PostHogProvider>;
}
