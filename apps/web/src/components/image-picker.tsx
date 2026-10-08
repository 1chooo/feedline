"use client";

import Image from "next/image";
import { useEffect, useRef, useState } from "react";

export function ImagePicker({ onValidityChange, onFileChange }: { onValidityChange: (valid: boolean) => void; onFileChange: (file: File | null) => void }) {
  const input = useRef<HTMLInputElement>(null);
  const objectURL = useRef<string | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => () => { if (objectURL.current) URL.revokeObjectURL(objectURL.current); }, []);

  function choose(file?: File) {
    if (objectURL.current) URL.revokeObjectURL(objectURL.current);
    objectURL.current = null;
    setPreview(null);
    const problem = file && (!/^(image\/(jpeg|png|webp|gif))$/.test(file.type) || file.size > 5 * 1024 * 1024)
      ? "Choose a JPEG, PNG, WebP, or GIF image of 5 MB or less." : null;
    setError(problem);
    onValidityChange(!problem);
    onFileChange(file && !problem ? file : null);
    if (file && !problem) {
      objectURL.current = URL.createObjectURL(file);
      setPreview(objectURL.current);
    }
  }

  function clear() {
    if (input.current) input.current.value = "";
    choose();
  }

  return <div className="mt-4 rounded-xl border border-border bg-surface p-3">
    <label htmlFor="post-image" className="block text-sm font-medium">Add a photo <span className="font-normal text-muted">(optional)</span></label>
    <input ref={input} id="post-image" name="image" type="file" accept="image/jpeg,image/png,image/webp,image/gif" aria-invalid={Boolean(error)} aria-describedby="post-image-help" onChange={(event) => choose(event.target.files?.[0])} className="mt-2 block w-full text-sm text-muted file:mr-3 file:rounded-full file:border-0 file:bg-background file:px-3 file:py-2 file:text-sm file:font-medium file:text-foreground" />
    <p id="post-image-help" className="mt-2 text-xs text-muted">JPEG, PNG, WebP, or GIF · up to 5 MB. Your photo is uploaded when you publish.</p>
    {preview ? <Image src={preview} alt="Selected photo preview" width={800} height={800} unoptimized className="mt-3 max-h-64 w-full rounded-lg object-contain" /> : null}
    {error ? <p role="alert" className="mt-2 text-sm text-red-600 dark:text-red-400">{error}</p> : null}
    {preview || error ? <button type="button" onClick={clear} className="mt-2 rounded-lg px-2 py-1 text-xs font-medium underline">Remove photo</button> : null}
  </div>;
}
