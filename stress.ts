#!/usr/bin/env -S deno run -A

// @ts-types="npm:@types/prompts@2.4.9"
import prompts from "npm:prompts@2.4.2";
import { findFreePorts } from "npm:find-free-ports@3.1.1";
import boxen from "npm:boxen@8.0.1";

console.log(`
NSQLite Stress Tester

This script will bombard the NSQLite server with a lot of queries to test it's performance.
`);

const ciMode = Deno.args.length > 0 && Deno.args[0] == "--ci";
let connections = 250;
let durationSeconds = 30;

if (!ciMode) {
  const responses = await prompts([
    {
      type: "number",
      name: "connections",
      message: "How many HTTP connections to use?",
      initial: 250,
    },
    {
      type: "number",
      name: "durationSeconds",
      message: "How many seconds to run the test?",
      initial: 30,
    },
  ], {
    onCancel: () => {
      Deno.exit(0);
    },
  });

  connections = responses.connections;
  durationSeconds = responses.durationSeconds;
  console.log();
}

const nsqlited = await spawnNsqlited();

await runQuery({
  baseUrl: nsqlited.baseUrl,
  query:
    "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);",
});

runBombardier({
  baseUrl: nsqlited.baseUrl,
  connections,
  durationSeconds,
  query: "INSERT INTO users (name, email) VALUES ('test', 'test@example.com');",
});

runBombardier({
  baseUrl: nsqlited.baseUrl,
  connections,
  durationSeconds,
  query: "SELECT * FROM users LIMIT 1;",
});

runBombardier({
  baseUrl: nsqlited.baseUrl,
  connections,
  durationSeconds,
  query: "SELECT 1, 2, 3;",
});

nsqlited.killProcess();

async function runQuery(opts: {
  baseUrl: string;
  query: string;
}) {
  const response = await fetch(`${opts.baseUrl}/query`, {
    method: "POST",
    body: JSON.stringify([{ query: opts.query }]),
  });
  const status = response.status;
  if (status !== 200) {
    throw new Error(`Query failed with status ${status}`);
  }
  console.log(`Query executed: ${opts.query}`);
}

function runBombardier(opts: {
  baseUrl: string;
  query: string;
  connections: number;
  durationSeconds: number;
}) {
  console.log(
    `\nBombarding for ${opts.durationSeconds} seconds`,
  );
  console.log(`Connections: ${opts.connections}`);
  console.log(`URL: ${opts.baseUrl}/query`);
  console.log(`Query: ${opts.query}`);

  // Docs: https://pkg.go.dev/github.com/codesenberg/bombardier
  const command = new Deno.Command("bombardier", {
    args: [
      "--print",
      "r",
      "--fasthttp",
      "--connections",
      opts.connections.toString(),
      "--duration",
      `${opts.durationSeconds}s`,
      "--body",
      JSON.stringify([{ query: opts.query }]),
      `${opts.baseUrl}/query`,
    ],
  });

  const output = command.outputSync();
  const outputString = new TextDecoder().decode(output.stdout);
  console.log(boxen(outputString, { padding: 1 }));
}

async function spawnNsqlited() {
  const [freePort] = await findFreePorts(1, { startPort: 10000 });
  const tempDir = Deno.makeTempDirSync({ prefix: "nsqlite_bombard_" });

  const command = new Deno.Command("nsqlited", {
    args: [
      "--listen-port",
      freePort.toString(),
      "--data-dir",
      tempDir,
    ],
    stdout: "null",
    stderr: "piped",
  });

  const process = command.spawn();
  const baseUrl = `http://localhost:${freePort}`;
  console.log(`Temporary NSQLite server running on ${baseUrl}`);

  await new Promise((resolve) => setTimeout(resolve, 2000));

  return {
    baseUrl,
    killProcess: () => {
      process.kill();
      Deno.removeSync(tempDir, { recursive: true });
    },
  };
}
