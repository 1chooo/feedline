"use client";

import { useActionState, useEffect, useRef } from "react";
import { createPostAction, type ComposeState } from "@/app/actions/posts";

const initialState: ComposeState = {};

export function ComposeForm() {
  const [state, action, pending] = useActionState(createPostAction, initialState);
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (state.ok) {
      formRef.current?.reset();
    }
  }, [state.ok]);

  return (
    <form
      ref={formRef}
      action={action}
      className="mx-auto w-full max-w-lg border-b border-border px-4 py-4"
    >
      <label className="sr-only" htmlFor="compose-title">
        Title
      </label>
      <input
        id="compose-title"
        name="title"
        required
        placeholder="Share something"
        className="w-full bg-transparent text-base font-medium outline-none placeholder:text-muted"
      />
      <textarea
        name="description"
        rows={3}
        placeholder="Add a caption"
        className="mt-2 w-full resize-none bg-transparent text-sm leading-6 outline-none placeholder:text-muted"
      />
      <input
        name="imageUrl"
        type="url"
        placeholder="Image URL (optional)"
        className="mt-2 w-full bg-transparent text-sm outline-none placeholder:text-muted"
      />
      <input
        name="landingPageUrl"
        type="url"
        placeholder="Link (optional)"
        className="mt-2 w-full bg-transparent text-sm outline-none placeholder:text-muted"
      />
      {state.error ? (
        <p className="mt-3 text-sm text-red-500">{state.error}</p>
      ) : null}
      <div className="mt-3 flex justify-end">
        <button
          type="submit"
          disabled={pending}
          className="rounded-full bg-foreground px-4 py-1.5 text-sm font-medium text-background disabled:opacity-60"
        >
          {pending ? "Posting…" : "Post"}
        </button>
      </div>
    </form>
  );
}
