import { type Client, NewClient } from "$lib/client/client";

const TOKEN_KEY = "nsqlite_token";
const BASE_URL_KEY = "nsqlite_baseUrl";

function defaultBaseUrl(): string {
  if (typeof window !== "undefined" && window.location?.origin) {
    return window.location.origin;
  }
  return "";
}

function stripRpcSuffix(url: string): string {
  return url
    .trim()
    .replace(/\/+rpc\/?$/, "")
    .replace(/\/+$/, "");
}

function rpcUrl(baseUrl: string): string {
  return `${baseUrl.replace(/\/+$/, "")}/rpc`;
}

class WebAppStore {
  token = $state("");
  baseUrl = $state("");
  client = $state<Client | null>(null);
  error = $state("");
  loaded = $state(false);

  constructor() {
    if (typeof localStorage !== "undefined") {
      this.token = localStorage.getItem(TOKEN_KEY) ?? "";
      this.baseUrl = stripRpcSuffix(
        localStorage.getItem(BASE_URL_KEY) ?? defaultBaseUrl(),
      );
      if (this.token && this.baseUrl) {
        this.buildClient();
      }
    }
    this.loaded = true;
  }

  private buildClient() {
    try {
      this.client = NewClient(rpcUrl(this.baseUrl))
        .withGlobalHeader("authorization", `Bearer ${this.token}`)
        .withGlobalTimeoutConfig({ timeoutMs: 10000 })
        .build();
      this.error = "";
    } catch (e) {
      this.error = String(e);
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(TOKEN_KEY, token);
    }
    if (this.baseUrl) {
      this.buildClient();
    }
  }

  setBaseUrl(url: string) {
    this.baseUrl = stripRpcSuffix(url);
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(BASE_URL_KEY, this.baseUrl);
    }
    if (this.token) {
      this.buildClient();
    }
  }

  verifyToken(baseUrl: string, token: string): Promise<void> {
    const url = stripRpcSuffix(baseUrl);
    this.error = "";
    return new Promise((resolve, reject) => {
      const tempClient = NewClient(rpcUrl(url))
        .withGlobalHeader("authorization", `Bearer ${token}`)
        .withGlobalTimeoutConfig({ timeoutMs: 10000 })
        .build();
      tempClient.procs
        .systemSession()
        .execute()
        .then((result) => {
          if (result.role) {
            this.setBaseUrl(url);
            this.setToken(token);
            resolve();
          } else {
            reject(new Error("Invalid token"));
          }
        })
        .catch((e) => {
          reject(e);
        });
    });
  }

  logout() {
    this.token = "";
    this.client = null;
    this.error = "";
    if (typeof localStorage !== "undefined") {
      localStorage.removeItem(TOKEN_KEY);
    }
  }
}

export const store = new WebAppStore();
