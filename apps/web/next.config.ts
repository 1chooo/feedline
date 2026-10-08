import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Docker Desktop on macOS often drops filesystem events. Turbopack polls
  // instead of relying on WATCHPACK_POLLING, which only affects webpack.
  watchOptions: {
    pollIntervalMs: 1000,
  },
  experimental: {
    serverActions: {
      bodySizeLimit: "6mb",
    },
  },
  async rewrites() {
    return [{ source: "/@:username", destination: "/users/:username" }];
  },
};

export default nextConfig;
