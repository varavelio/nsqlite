<script lang="ts">
  import { Alert, Badge, Button, Card, Table, Textarea } from "@varavel/ui";
  import type { QueryResult, SqliteValue } from "$lib/client/types";
  import { store } from "$lib/store.svelte";

  let sql = $state("");
  let results = $state<QueryResult[]>([]);
  let totalTime = $state(0);
  let error = $state("");
  let running = $state(false);

  function sqliteValueToDisplay(v: SqliteValue): string {
    if (v.null === true) return "NULL";
    if (v.integer !== undefined) return String(v.integer);
    if (v.real !== undefined) return String(v.real);
    if (v.text !== undefined) return v.text;
    if (v.blob !== undefined) return `[BLOB: ${v.blob.length} chars]`;
    return "";
  }

  async function execute() {
    if (!store.client || !sql.trim()) return;

    error = "";
    results = [];
    running = true;

    try {
      const output = await store.client.procs.databaseQuery().execute({
        queries: [{ query: sql.trim() }],
      });
      totalTime = output.time;
      results = output.results;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      running = false;
    }
  }

  function resultBadgeColor(type: string) {
    switch (type) {
      case "read":
        return "info";
      case "write":
        return "warning";
      case "begin":
        return "neutral";
      case "commit":
        return "success";
      case "rollback":
        return "warning";
      case "error":
        return "error";
      default:
        return "neutral";
    }
  }
</script>

<div class="flex flex-col gap-4">
  <Card class="p-4">
    <div class="flex flex-col gap-3">
      <label for="sql-editor" class="text-sm font-medium">SQL Query</label>
      <Textarea
        id="sql-editor"
        placeholder="SELECT * FROM sqlite_master;"
        bind:value={sql}
        rows={5}
      />
      <div class="flex items-center gap-3">
        <Button
          color="info"
          size="md"
          loading={running}
          disabled={running || !sql.trim()}
          onclick={execute}
        >
          Execute
        </Button>
        {#if !running && results.length > 0}
          <span class="text-xs text-(--color-text-muted)">
            {results.length} result{results.length !== 1 ? "s" : ""} in {totalTime.toFixed(3)}
            s
          </span>
        {/if}
      </div>
    </div>
  </Card>

  {#if error}
    <Alert color="error">{error}</Alert>
  {/if}

  {#each results as result}
    <Card class="p-4">
      <div class="flex items-center gap-2 mb-3">
        <Badge color={resultBadgeColor(result.type)} size="sm">
          {result.type}
        </Badge>
        <span class="text-xs text-(--color-text-muted)">
          {result.time.toFixed(4)}
          s
        </span>
        {#if result.txId}
          <span class="text-xs text-(--color-text-muted)">
            tx: {result.txId}
          </span>
        {/if}
        {#if result.lastInsertId !== undefined}
          <span class="text-xs text-(--color-text-muted)">
            lastId: {result.lastInsertId}
          </span>
        {/if}
        {#if result.rowsAffected !== undefined}
          <span class="text-xs text-(--color-text-muted)">
            rows: {result.rowsAffected}
          </span>
        {/if}
      </div>

      {#if result.error}
        <Alert color="error">{result.error}</Alert>
      {/if}

      {#if result.columns && result.rows && result.columns.length > 0}
        <div class="overflow-x-auto">
          <Table.Root size="sm" striped stickyHeader>
            <Table.Header>
              <Table.Row>
                {#each result.columns as col}
                  <Table.Head>{col}</Table.Head>
                {/each}
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each result.rows as row}
                <Table.Row>
                  {#each row as cell}
                    <Table.Cell>{sqliteValueToDisplay(cell)}</Table.Cell>
                  {/each}
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
        </div>
      {:else if ["write", "begin", "commit", "rollback"].includes(result.type)}
        <p class="text-sm text-(--color-text-muted)">
          Statement executed successfully with no rows returned.
        </p>
      {/if}
    </Card>
  {/each}
</div>
