import Link from "next/link";
import type { ComponentProps } from "react";

type ProfileLinkProps = Omit<ComponentProps<typeof Link>, "href"> & { username: string };

export function ProfileLink({ username, ...props }: ProfileLinkProps) {
  return <Link href={`/users/${encodeURIComponent(username)}`} {...props} />;
}
