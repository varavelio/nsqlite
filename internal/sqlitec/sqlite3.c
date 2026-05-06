#ifndef NSQLITE_C_WRAPPER
#define NSQLITE_C_WRAPPER

#include "sqlite3-v3.48.0.h"
#include <stdlib.h>

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

#endif
