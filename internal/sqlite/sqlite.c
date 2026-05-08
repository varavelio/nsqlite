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
    "abs",
    "avg",
    "changes",
    "char",
    "coalesce",
    "concat",
    "concat_ws",
    "count",
    "date",
    "datetime",
    "format",
    "glob",
    "group_concat",
    "hex",
    "if",
    "ifnull",
    "iif",
    "instr",
    "json",
    "json_array",
    "json_array_insert",
    "json_array_length",
    "json_each",
    "json_error_position",
    "json_extract",
    "json_group_array",
    "json_group_object",
    "json_insert",
    "json_object",
    "json_patch",
    "json_pretty",
    "json_quote",
    "json_remove",
    "json_replace",
    "json_set",
    "json_tree",
    "json_type",
    "json_valid",
    "jsonb",
    "jsonb_array",
    "jsonb_array_insert",
    "jsonb_each",
    "jsonb_extract",
    "jsonb_group_array",
    "jsonb_group_object",
    "jsonb_insert",
    "jsonb_object",
    "jsonb_patch",
    "jsonb_remove",
    "jsonb_replace",
    "jsonb_set",
    "jsonb_tree",
    "julianday",
    "last_insert_rowid",
    "length",
    "like",
    "likelihood",
    "likely",
    "lower",
    "ltrim",
    "max",
    "min",
    "nullif",
    "octet_length",
    "printf",
    "quote",
    "random",
    "randomblob",
    "replace",
    "round",
    "rtrim",
    "sign",
    "sqlite_compileoption_get",
    "sqlite_compileoption_used",
    "sqlite_source_id",
    "sqlite_version",
    "strftime",
    "substr",
    "substring",
    "sum",
    "time",
    "timediff",
    "total",
    "total_changes",
    "trim",
    "typeof",
    "unhex",
    "unicode",
    "unistr",
    "unistr_quote",
    "unixepoch",
    "unlikely",
    "upper",
    "zeroblob",
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

// Read-only PRAGMAs that are safe for READ_WRITE and READ_ONLY roles,
// categorised by whether they accept a first argument (arg4).
//
// Introspection pragmas always need a table name argument (e.g.
//   PRAGMA table_info(users)
// returns 1 unconditionally.
//
// Stateless pragmas are genuinely read-only only when called without an
// assignment argument (arg4 == NULL).  Passing arg4 turns them into writes
// (e.g. PRAGMA user_version = 123).
//
// Expensive pragmas (integrity_check, quick_check, foreign_key_check) and
// state-changing pragmas are intentionally excluded — admin-only.
static int cust_authorizer_is_readonly_pragma(const char *name, const char *arg) {
  if (name == NULL) {
    return 0;
  }

  // Introspection pragmas. They are read-only even when they take a table/index argument.
  if (
      strcmp(name, "table_info") == 0 ||
      strcmp(name, "table_xinfo") == 0 ||
      strcmp(name, "index_info") == 0 ||
      strcmp(name, "index_xinfo") == 0 ||
      strcmp(name, "index_list") == 0 ||
      strcmp(name, "foreign_key_list") == 0
  ) {
    return 1;
  }

  // Stateless read-only pragmas — safe only when called without an argument.
  if (
      strcmp(name, "collation_list") == 0 ||
      strcmp(name, "compile_options") == 0 ||
      strcmp(name, "data_version") == 0 ||
      strcmp(name, "database_list") == 0 ||
      strcmp(name, "defer_foreign_keys") == 0 ||
      strcmp(name, "freelist_count") == 0 ||
      strcmp(name, "function_list") == 0 ||
      strcmp(name, "module_list") == 0 ||
      strcmp(name, "page_count") == 0 ||
      strcmp(name, "page_size") == 0 ||
      strcmp(name, "pragma_list") == 0 ||
      strcmp(name, "schema_version") == 0 ||
      strcmp(name, "user_version") == 0
  ) {
    return arg == NULL;
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
      return cust_authorizer_is_readonly_pragma(arg3, arg4) ? SQLITE_OK : SQLITE_DENY;

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
      return cust_authorizer_is_readonly_pragma(arg3, arg4) ? SQLITE_OK : SQLITE_DENY;
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
