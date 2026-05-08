<script>
  import "../app.css";

  import { Button, UiProvider } from "@varavel/ui";
  import { Loader } from "@varavel/ui/brand";
  import { AppLayout } from "@varavel/ui/layouts";
  import { store } from "$lib/store.svelte";
  import AuthWall from "./AuthWall.svelte";

  let { children } = $props();

  let initialized = $derived(store.loaded);

  function handleLogout() {
    store.logout();
  }
</script>

{#if !initialized}
  <div class="w-dvw h-dvh overflow-hidden flex items-center justify-center">
    <Loader size="lg" />
  </div>
{:else}
  <UiProvider>
    <AuthWall>
      <AppLayout primaryRegion="header" maxWidth="md">
        {#snippet headerLeft()}
          <header class="hidden desk:flex ml-2 items-center gap-4">
            <img
              src="https://cdn.jsdelivr.net/gh/varavelio/nsqlite@dbf7ff/assets/logo.svg"
              alt="NSQLite Logo"
              class="h-6 mx-auto"
            >
          </header>
        {/snippet}

        {#snippet headerCenter()}
          <header class="w-full desk:hidden"></header>
        {/snippet}

        {#snippet headerRight()}
          {#if store.client}
            <Button
              variant="ghost"
              size="sm"
              color="neutral"
              onclick={handleLogout}
            >
              Logout
            </Button>
          {/if}
        {/snippet}

        {#snippet main()}
          {@render children()}
        {/snippet}
      </AppLayout>
    </AuthWall>
  </UiProvider>
{/if}
