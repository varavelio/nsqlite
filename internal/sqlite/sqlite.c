#ifndef NSQLITE_C_WRAPPER
#define NSQLITE_C_WRAPPER

#include "sqlite3.h"
#include <stdlib.h>
#include <string.h>

// Here we define some C functions that are needed to access some of the
// SQLite3 C API functions that are not directly accessible from Go and
// other utilities that are useful for this project.

// SQLITE_TRANSIENT is not accessible from Go, so we create a wrapper here.
static int cust_sqlite3_bind_text(sqlite3_stmt *stmt, int n, char *p, int np) {
  return sqlite3_bind_text(stmt, n, p, np, SQLITE_TRANSIENT);
}

// SQLITE_TRANSIENT is not accessible from Go, so we create a wrapper here.
static int cust_sqlite3_bind_blob(sqlite3_stmt *stmt, int n, void *p, int np) {
  return sqlite3_bind_blob(stmt, n, p, np, SQLITE_TRANSIENT);
}

typedef struct {
  int role;
} cust_authorizer_ctx;

enum {
  CUST_AUTHORIZER_ROLE_ADMIN = 0,
  CUST_AUTHORIZER_ROLE_READ_WRITE = 1,
  CUST_AUTHORIZER_ROLE_READ_ONLY = 2,
};

static int cust_authorizer_is_internal_table(const char *name) {
  return name != NULL && strncmp(name, "sqlite_", 7) == 0;
}

// Built-in scalar and aggregate functions that are safe for READ_WRITE and
// READ_ONLY roles.  All items are side-effect-free from a database-modification
// perspective.  The only notable omission is load_extension(), which can load
// arbitrary shared libraries.
static const char *allowed_function_names[] = {
    "abs",                       "avg",
    "coalesce",                  "concat",
    "date",                      "datetime",
    "group_concat",              "hex",
    "instr",                     "json",
    "json_each",                 "json_extract",
    "json_insert",               "json_object",
    "json_remove",               "json_replace",
    "json_valid",                "julianday",
    "like",                      "likelihood",
    "ltrim",                     "max",
    "octet_length",              "printf",
    "randomblob",                "replace",
    "changes",                   "char",
    "concat_ws",                 "count",
    "format",                    "glob",
    "ifnull",                    "iif",
    "json_array",                "json_array_length",
    "json_group_array",          "json_group_object",
    "json_patch",                "json_quote",
    "json_set",                  "json_type",
    "last_insert_rowid",         "length",
    "likely",                    "lower",
    "min",                       "nullif",
    "quote",                     "random",
    "round",                     "rtrim",
    "sign",                      "sqlite_compileoption_get",
    "sqlite_compileoption_used", "sqlite_offset",
    "sqlite_version",            "strftime",
    "sum",                       "time",
    "total_changes",             "trim",
    "unicode",                   "unlikely",
    "substr",                    "substring",
    "timediff",                  "total",
    "typeof",                    "unhex",
    "upper",                     "zeroblob",
    "sqlite_source_id",
    NULL
};

static int cust_authorizer_is_allowed_function(const char *name) {
  if (name == NULL) {
    return 0;
  }
  for (int i = 0; allowed_function_names[i] != NULL; i++) {
    if (strcmp(name, allowed_function_names[i]) == 0) {
      return 1;
    }
  }
  return 0;
}

// Read-only PRAGMAs that are safe for READ_WRITE and READ_ONLY roles. PRAGMAs
// that change database state (journal_mode, synchronous, foreign_keys, …) are
// intentionally excluded.
static const char *readonly_pragma_names[] = {
    "collation_list",   "compile_options",    "data_version",
    "database_list",    "defer_foreign_keys", "foreign_key_check",
    "foreign_key_list", "freelist_count",     "function_list",
    "index_info",       "index_list",         "index_xinfo",
    "integrity_check",  "module_list",        "page_count",
    "page_size",        "pragma_list",        "quick_check",
    "schema_version",   "table_info",         "table_xinfo",
    "user_version",
    NULL
};

static int cust_authorizer_is_readonly_pragma(const char *name) {
  if (name == NULL) {
    return 0;
  }
  for (int i = 0; readonly_pragma_names[i] != NULL; i++) {
    if (strcmp(name, readonly_pragma_names[i]) == 0) {
      return 1;
    }
  }
  return 0;
}

static int cust_sqlite3_authorizer(void *pUserData, int action, const char *arg3,
                                   const char *arg4, const char *arg5, const char *arg6) {
  (void)arg5;
  (void)arg6;

  cust_authorizer_ctx *ctx = (cust_authorizer_ctx *)pUserData;
  if (ctx == NULL) {
    return SQLITE_DENY;
  }

  switch (ctx->role) {
  case CUST_AUTHORIZER_ROLE_ADMIN:
    return SQLITE_OK;

  case CUST_AUTHORIZER_ROLE_READ_WRITE:
    switch (action) {
    case SQLITE_SELECT:
    case SQLITE_READ:
    case SQLITE_RECURSIVE:
      return SQLITE_OK;

    case SQLITE_FUNCTION:
      return cust_authorizer_is_allowed_function(arg4) ? SQLITE_OK : SQLITE_DENY;

    case SQLITE_PRAGMA:
      return cust_authorizer_is_readonly_pragma(arg3) ? SQLITE_OK : SQLITE_DENY;

    case SQLITE_TRANSACTION:
    case SQLITE_SAVEPOINT:
      return SQLITE_OK;

    case SQLITE_INSERT:
    case SQLITE_UPDATE:
    case SQLITE_DELETE:
      if (cust_authorizer_is_internal_table(arg3)) {
        return SQLITE_DENY;
      }
      return SQLITE_OK;
    }
    return SQLITE_DENY;

  case CUST_AUTHORIZER_ROLE_READ_ONLY:
    switch (action) {
    case SQLITE_SELECT:
    case SQLITE_READ:
    case SQLITE_RECURSIVE:
      return SQLITE_OK;

    case SQLITE_FUNCTION:
      return cust_authorizer_is_allowed_function(arg4) ? SQLITE_OK : SQLITE_DENY;

    case SQLITE_PRAGMA:
      return cust_authorizer_is_readonly_pragma(arg3) ? SQLITE_OK : SQLITE_DENY;
    }
    return SQLITE_DENY;
  }

  return SQLITE_DENY;
}

static cust_authorizer_ctx *cust_authorizer_ctx_new(void) {
  cust_authorizer_ctx *ctx = calloc(1, sizeof(cust_authorizer_ctx));
  if (ctx == NULL) {
    return NULL;
  }
  ctx->role = CUST_AUTHORIZER_ROLE_ADMIN;
  return ctx;
}

static void cust_authorizer_ctx_free(cust_authorizer_ctx *ctx) {
  free(ctx);
}

static int cust_authorizer_ctx_set_role(cust_authorizer_ctx *ctx, int role) {
  if (ctx == NULL) {
    return SQLITE_MISUSE;
  }
  ctx->role = role;
  return SQLITE_OK;
}

static int cust_sqlite3_set_authorizer(sqlite3 *db, cust_authorizer_ctx *ctx) {
  return sqlite3_set_authorizer(db, cust_sqlite3_authorizer, ctx);
}

#endif
