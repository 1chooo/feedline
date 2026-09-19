"use client";

import type { ComponentProps, MouseEvent } from "react";

type ProfileLinkProps = Omit<ComponentProps<"a">, "href"> & {
  username: string;
};

export function ProfileLink({ username, onClick, ...props }: ProfileLinkProps) {
  const href = `/@${username}`;

  function handleClick(event: MouseEvent<HTMLAnchorElement>) {
    onClick?.(event);
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return;
    }

    event.preventDefault();
    window.location.assign(href);
  }

  return <a href={href} onClick={handleClick} {...props} />;
}
