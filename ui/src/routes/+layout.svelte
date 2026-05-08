<script>
  import "../app.css";

  import { UiProvider } from "@varavel/ui";

  let { children } = $props();
</script>

<UiProvider>{@render children()}</UiProvider>
