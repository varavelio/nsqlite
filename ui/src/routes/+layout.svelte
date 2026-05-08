<script>
  import "../app.css";

  import { UiProvider } from "@varavel/ui";
  import { Loader } from "@varavel/ui/brand";
  import { AppLayout } from "@varavel/ui/layouts";

  let { children } = $props();

  let initialized = $state(false);
</script>

{#if !initialized}
  <div class="w-dvw h-dvh overflow-hidden flex items-center justify-center">
    <Loader size="lg" />
  </div>
{:else}
  <UiProvider>
    <AppLayout primaryRegion="header" maxWidth="md">
      {#snippet headerLeft()}
        <header class="hidden desk:flex ml-2 items-center gap-4"></header>
      {/snippet}

      {#snippet headerCenter()}
        <header class="w-full desk:hidden"></header>
      {/snippet}

      {#snippet headerRight()}
      {/snippet}

      {#snippet sidebarTop()}
      {/snippet}

      {#snippet sidebarCenter()}
      {/snippet}

      {#snippet main()}
        {@render children()}
      {/snippet}
    </AppLayout>
  </UiProvider>
{/if}
