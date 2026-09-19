import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [{ source: "/@:username", destination: "/users/:username" }];
  },
};

export default nextConfig;
