<script lang="ts">
  import { Button, Card, Input } from "@varavel/ui";
  import { toast } from "@varavel/ui/runtime";
  import { store } from "$lib/store.svelte";

  let { children } = $props();

  let dsnInput = $state("");
  let loading = $state(false);

  async function verify() {
    const trimmed = dsnInput.trim();

    if (!trimmed) {
      toast.error("DSN is required");
      return;
    }

    loading = true;

    try {
      await store.verifyDsn(trimmed);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      loading = false;
    }
  }
</script>

{#if store.client}
  {@render children()}
{:else}
  <div class="w-dvw h-dvh overflow-hidden flex items-center justify-center p-4">
    <Card class="w-full max-w-md">
      <div class="flex flex-col gap-6">
        <div class="flex flex-col gap-2 text-center">
          <img
            src="https://cdn.jsdelivr.net/gh/varavelio/nsqlite@dbf7ff/assets/logo.svg"
            alt="NSQLite Logo"
            class="h-10 mx-auto"
          >
          <p class="text-sm text-(--color-text-muted)">
            Connect to your NSQLite server
          </p>
        </div>

        <form onsubmit={(e) => { e.preventDefault(); verify(); }}>
          <div class="flex flex-col gap-4">
            <div class="flex flex-col gap-1">
              <label for="dsn" class="text-xs font-medium">
                DSN
                <span class="text-(--color-text-muted) font-normal">
                  (Data Source Name)
                </span>
              </label>
              <Input
                id="dsn"
                placeholder="http(s)://host:port?authToken=..."
                bind:value={dsnInput}
                size="md"
              />
            </div>

            <Button
              type="submit"
              color="info"
              variant="solid"
              size="md"
              wide
              {loading}
              disabled={loading}
            >
              Connect
            </Button>
          </div>
        </form>
      </div>
    </Card>
  </div>
{/if}
