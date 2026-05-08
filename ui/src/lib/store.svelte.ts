import { type Client, NewClient } from "$lib/client/client";
import { type ParsedDsn, parseDsn, rpcUrl } from "$lib/dsn";

const DSN_KEY = "nsqlite_dsn";

class WebAppStore {
  dsn = $state("");
  parsed = $state<ParsedDsn | null>(null);
  client = $state<Client | null>(null);
  error = $state("");
  loaded = $state(false);

  constructor() {
    if (typeof localStorage !== "undefined") {
      this.dsn = localStorage.getItem(DSN_KEY) ?? "";
    }
    if (this.dsn) {
      this.parsed = parseDsn(this.dsn);
      if (this.parsed.host) {
        this.buildClient();
      }
    }
    this.loaded = true;
  }

  private buildClient() {
    if (!this.parsed) return;
    try {
      const builder = NewClient(rpcUrl(this.parsed));
      if (this.parsed.authToken) {
        builder.withGlobalHeader(
          "authorization",
          `Bearer ${this.parsed.authToken}`,
        );
      }
      this.client = builder
        .withGlobalTimeoutConfig({ timeoutMs: 10000 })
        .build();
      this.error = "";
    } catch (e) {
      this.error = String(e);
    }
  }

  setDsn(dsn: string) {
    this.dsn = dsn;
    this.parsed = parseDsn(dsn);
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(DSN_KEY, dsn);
    }
    if (this.parsed.host) {
      this.buildClient();
    }
  }

  verifyDsn(dsn: string): Promise<void> {
    const parsed = parseDsn(dsn);
    this.error = "";
    return new Promise((resolve, reject) => {
      const builder = NewClient(rpcUrl(parsed));
      if (parsed.authToken) {
        builder.withGlobalHeader("authorization", `Bearer ${parsed.authToken}`);
      }
      const tempClient = builder
        .withGlobalTimeoutConfig({ timeoutMs: 10000 })
        .build();
      tempClient.procs
        .systemSession()
        .execute()
        .then((result) => {
          if (result.role) {
            this.setDsn(dsn);
            resolve();
          } else {
            reject(new Error("Invalid session"));
          }
        })
        .catch((e) => {
          reject(e);
        });
    });
  }

  logout() {
    this.dsn = "";
    this.client = null;
    this.error = "";
    this.parsed = null;
    if (typeof localStorage !== "undefined") {
      localStorage.removeItem(DSN_KEY);
    }
  }
}

export const store = new WebAppStore();
