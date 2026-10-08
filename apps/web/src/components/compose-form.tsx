"use client";

import { useActionState, useEffect, useRef, useState } from "react";
import { ImagePicker } from "@/components/image-picker";
import { createPostAction, type ComposeState } from "@/app/actions/posts";

const initialState: ComposeState = {};

export function ComposeForm() {
  const formRef = useRef<HTMLFormElement>(null);
  const [imageValid, setImageValid] = useState(true);
  const [image, setImage] = useState<File | null>(null);
  const emptyFields = { title: "", description: "", imageUrl: "", landingPageUrl: "" };
  const [fields, setFields] = useState(emptyFields);
  const [state, action, pending] = useActionState(async (previous: ComposeState, data: FormData) => {
    if (image) data.set("image", image);
    const result = await createPostAction(previous, data);
    if (result.ok) { setFields(emptyFields); setImage(null); }
    return result;
  }, initialState);

  useEffect(() => {
    if (state.ok) {
      formRef.current?.reset();
    }
  }, [state.submissionId, state.ok]);

  return (
    <form
      ref={formRef}
      action={action}
      className="mx-auto w-full max-w-lg border-b border-border px-4 py-4"
    >
      <fieldset disabled={pending}>
      <label className="mb-2 block text-sm font-medium" htmlFor="compose-title">
        Share with the community
      </label>
      <input
        id="compose-title"
        name="title"
        value={fields.title}
        onChange={(event) => setFields({ ...fields, title: event.target.value })}
        required
        maxLength={200}
        placeholder="Share something"
        className="w-full bg-transparent text-base font-medium outline-none placeholder:text-muted"
      />
      <textarea
        aria-label="Post caption"
        name="description"
        value={fields.description}
        onChange={(event) => setFields({ ...fields, description: event.target.value })}
        rows={3}
        maxLength={2000}
        placeholder="Add a caption"
        className="mt-2 w-full resize-none bg-transparent text-sm leading-6 outline-none placeholder:text-muted"
      />
      <ImagePicker key={state.submissionId ?? "new-post"} onValidityChange={setImageValid} onFileChange={setImage} />
      <details className="mt-3 text-sm">
      <summary className="cursor-pointer font-medium text-muted">Add links (optional)</summary>
      <input
        name="imageUrl"
        value={fields.imageUrl}
        onChange={(event) => setFields({ ...fields, imageUrl: event.target.value })}
        aria-label="External image URL"
        type="url"
        placeholder="Image URL (optional)"
        className="mt-2 w-full bg-transparent text-sm outline-none placeholder:text-muted"
      />
      <input
        name="landingPageUrl"
        value={fields.landingPageUrl}
        onChange={(event) => setFields({ ...fields, landingPageUrl: event.target.value })}
        aria-label="Post link"
        type="url"
        placeholder="Link (optional)"
        className="mt-2 w-full bg-transparent text-sm outline-none placeholder:text-muted"
      />
      </details>
      </fieldset>
      {state.error ? (
        <p role="alert" className="mt-3 text-sm text-red-500">{state.error}</p>
      ) : null}
      {state.ok ? <p role="status" className="mt-3 text-sm text-green-600">Your post is published.</p> : null}
      <div className="mt-3 flex justify-end">
        <button
          type="submit"
          disabled={pending || !imageValid}
          className="rounded-full bg-foreground px-4 py-1.5 text-sm font-medium text-background disabled:opacity-60"
        >
          {pending ? "Posting…" : "Post"}
        </button>
      </div>
    </form>
  );
}
