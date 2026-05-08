<script lang="ts">
  import { Tabs } from "@varavel/ui";
  import SqlQuery from "./SqlQuery.svelte";
  import SystemStatus from "./SystemStatus.svelte";

  let activeTab = $state("status");
  let tabs = $state([
    { value: "status", label: "System Status", content: statusSnippet },
    { value: "query", label: "SQL Query", content: querySnippet },
  ]);
</script>

{#snippet statusSnippet()}
  <SystemStatus />
{/snippet}

{#snippet querySnippet()}
  <SqlQuery />
{/snippet}

<div class="p-4">
  <Tabs bind:value={activeTab} items={tabs} color="info" />
</div>
