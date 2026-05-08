interface ParsedDsn {
  protocol: string;
  host: string;
  port: string;
  authToken: string;
}

function parseDsn(dsn: string): ParsedDsn {
  const trimmed = dsn.trim();

  let protocol = "http";
  let host = "";
  let port = "9876";
  let authToken = "";

  let rest = trimmed;
  if (rest.includes("://")) {
    const idx = rest.indexOf("://");
    protocol = rest.slice(0, idx) || "http";
    rest = rest.slice(idx + 3);
  }

  if (protocol !== "http" && protocol !== "https") {
    protocol = "http";
  }

  const queryIdx = rest.indexOf("?");
  if (queryIdx >= 0) {
    const queryPart = rest.slice(queryIdx + 1);
    rest = rest.slice(0, queryIdx);
    const params = new URLSearchParams(queryPart);
    authToken = params.get("authToken") ?? "";
  }

  if (rest.includes(":")) {
    const colonIdx = rest.lastIndexOf(":");
    host = rest.slice(0, colonIdx);
    port = rest.slice(colonIdx + 1) || "9876";
  } else {
    host = rest;
  }

  return { protocol, host, port: port || "9876", authToken };
}

function baseUrl(d: ParsedDsn): string {
  return `${d.protocol}://${d.host}:${d.port}`;
}

function rpcUrl(d: ParsedDsn): string {
  return `${baseUrl(d)}/rpc`;
}

export type { ParsedDsn };
export { baseUrl, parseDsn, rpcUrl };
