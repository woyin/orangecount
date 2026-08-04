<script lang="ts">
  import { pageLabel } from "../router.mjs";

  export let route: string;
  export let account = "";

  $: title = route === "account" && account ? account : pageLabel(route);
</script>

<h1 id="page-title">{title}</h1>
