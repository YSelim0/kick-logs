/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "files.kick.com"
      },
      {
        protocol: "https",
        hostname: "kick.com"
      }
    ]
  }
};

export default nextConfig;
