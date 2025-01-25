#!/usr/bin/env -S deno run -A

const targets = [
  { os: "linux", goarch: "amd64", zigarch: "x86_64-linux-musl" },
  { os: "linux", goarch: "arm64", zigarch: "aarch64-linux-musl" },
  { os: "windows", goarch: "amd64", zigarch: "x86_64-windows-gnu" },
  { os: "windows", goarch: "arm64", zigarch: "aarch64-windows-gnu" },
  // TODO: Fix macos build error
  // error: unable to find dynamic system library 'resolv' using strategy 'paths_first'. searched paths: none
  // { os: "darwin", goarch: "amd64", zigarch: "x86_64-macos-none" },
  // { os: "darwin", goarch: "arm64", zigarch: "aarch64-macos-none" },
];

const cmds = [
  "nsqlite",
  "nsqlited",
  "nsqlitebench",
];

const buildsQty = targets.length * cmds.length;
let buildsCount = 0;

for await (const target of targets) {
  const env = {
    CGO_ENABLED: "1",
    GOOS: target.os,
    GOARCH: target.goarch,
    CC: `zig cc -target ${target.zigarch}`,
    CXX: `zig c++ -target ${target.zigarch}`,
  };

  for await (const cmd of cmds) {
    const srcPath = `./cmd/${cmd}/.`;
    let destPath = `./dist/${target.os}-${target.goarch}/${cmd}`;
    if (target.os === "windows") destPath += ".exe";

    buildsCount++;
    print(`${buildsCount}/${buildsQty} Building ${destPath}`);

    const args = [
      "build",
      "-o",
      destPath,
      srcPath,
    ];

    const c = new Deno.Command("go", { args, env });
    const { success, stderr } = await c.output();
    if (!success) {
      print("\n");
      console.error(new TextDecoder().decode(stderr));
      Deno.exit(1);
    }

    print(" -> OK\n");
  }

  const srcPath = `./dist/${target.os}-${target.goarch}/*`;
  const zipPath = `./dist/${target.os}-${target.goarch}.zip`;
  const zip = new Deno.Command("7z", { args: ["a", zipPath, srcPath] });
  await zip.output();
}

async function print(text: string) {
  await Deno.stdout.write(new TextEncoder().encode(text));
}
