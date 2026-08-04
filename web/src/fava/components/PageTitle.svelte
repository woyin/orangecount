<script lang="ts">
  import { pageLabel } from "../router.mjs";

  export let route: string;
  export let account = "";

  $: title = route === "account" && account ? account : pageLabel(route);
</script>

<strong id="page-title">{title}</strong>

<style>
  strong::before {
    margin: 0 10px;
    font-weight: normal;
    content: "›";
    opacity: 0.5;
  }
</style>
