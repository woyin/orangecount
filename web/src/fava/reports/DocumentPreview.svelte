<!-- This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. -->

<!--
  Inline preview for a selected document, mirroring Fava's DocumentPreview:
  PDFs in an <object>, images in an <img>, plain-text formats fetched and
  shown read-only, HTML sandboxed. Upstream renders the plain-text reader
  with CodeMirror; the transplanted shell shows a <pre> until H1 lands.
-->
<script lang="ts">
  import { translations, type Locale } from "../../translations";

  export let filename: string;
  export let locale = "en";

  const plainTextExtensions = ["csv", "json", "qfx", "txt", "xml"];
  const imageExtensions = ["gif", "jpg", "jpeg", "png", "svg", "webp", "bmp", "ico"];

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function segment(text: string): string {
    return text.split("/").map(encodeURIComponent).join("/");
  }

  $: extension = (filename.split(".").pop() ?? "").toLowerCase();
  $: url = `/documents/${segment(filename)}`;

  let text: string | null = null;
  let textError: string | null = null;
  $: if (plainTextExtensions.includes(extension)) {
    text = null;
    textError = null;
    void loadText(url);
  }

  async function loadText(target: string) {
    try {
      const response = await fetch(target);
      if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
      text = await response.text();
    } catch (value) {
      textError = value instanceof Error ? value.message : String(value);
    }
  }

  $: notice = t("previewNotImplemented").replace("%(filename)s", filename).replace("%(extension)s", extension);
</script>

{#if extension === "pdf"}
  <object title={filename} data={url}></object>
{:else if plainTextExtensions.includes(extension)}
  {#if textError}
    <p class="error">{textError}</p>
  {:else if text != null}
    <pre class="text-preview">{text}</pre>
  {:else}
    <p>…</p>
  {/if}
{:else if imageExtensions.includes(extension)}
  <img src={url} alt={filename} />
{:else if extension === "html" || extension === "htm"}
  <iframe src={url} title={filename} sandbox=""></iframe>
{:else}
  <p>{notice}</p>
{/if}

<style>
  object,
  img,
  iframe,
  .text-preview {
    width: 100%;
    height: 100%;
  }

  img {
    object-fit: contain;
  }

  .text-preview {
    overflow: auto;
  }
</style>
