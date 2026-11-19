package env

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/exr462/go-dark/config"
)

// DBUser is a service's database identity the way the real database has
// it: the application user the service connects as, and the schema owner
// that runs Liquibase and holds the objects (the same user when the service
// owns its schema).
type DBUser struct {
	App, Owner string
}

var (
	datasourceUser = regexp.MustCompile(`(?m)^spring\.datasource\.username=([A-Za-z0-9_$#]+)[ \t\r]*$`)
	liquibaseUser  = regexp.MustCompile(`(?m)^spring\.liquibase\.user=([A-Za-z0-9_$#]+)[ \t\r]*$`)
)

// Users reads the database users of the services under projectDirs from
// their helm/*/config/<configFrom>/*.properties, adds the manifest's extra
// users and returns them sorted without duplicates.
func Users(projectDirs []string, configFrom string, extra []config.DatabaseUser) []DBUser {
	var users []DBUser
	for _, dir := range projectDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "helm", "*", "config", configFrom, "*.properties"))
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			app := datasourceUser.FindSubmatch(data)
			if app == nil {
				continue
			}
			u := DBUser{App: string(app[1])}
			if owner := liquibaseUser.FindSubmatch(data); owner != nil {
				u.Owner = string(owner[1])
			}
			users = append(users, u)
		}
	}
	for _, x := range extra {
		users = append(users, DBUser{App: x.App, Owner: x.Owner})
	}
	for i := range users {
		users[i].App = strings.ToUpper(users[i].App)
		users[i].Owner = strings.ToUpper(users[i].Owner)
		if users[i].Owner == "" {
			users[i].Owner = users[i].App
		}
	}
	slices.SortFunc(users, func(a, b DBUser) int {
		return strings.Compare(a.App+" "+a.Owner, b.App+" "+b.Owner)
	})
	return slices.Compact(users)
}

// UserNames lists every distinct user name (apps and owners), sorted.
func UserNames(users []DBUser) []string {
	var names []string
	for _, u := range users {
		for _, n := range []string{u.Owner, u.App} {
			if !slices.Contains(names, n) {
				names = append(names, n)
			}
		}
	}
	slices.Sort(names)
	return names
}

// SchemaSQL creates the users that don't exist yet and grants what they
// need: owners hold the objects Liquibase creates, application users reach
// them unqualified (a logon trigger sets their session schema to the owner)
// with DML rights. Idempotent. Fed to sqlplus as sysdba; it exits non-zero
// on the first error.
func SchemaSQL(users []DBUser, password string) string {
	var b strings.Builder
	b.WriteString("WHENEVER SQLERROR EXIT FAILURE\n")
	for _, u := range users {
		for _, name := range uniq(u.Owner, u.App) {
			fmt.Fprintf(&b, `DECLARE n NUMBER; BEGIN
  SELECT COUNT(*) INTO n FROM dba_users WHERE username = '%[1]s';
  IF n = 0 THEN EXECUTE IMMEDIATE 'CREATE USER %[1]s IDENTIFIED BY "%[2]s" QUOTA UNLIMITED ON USERS'; END IF;
  EXECUTE IMMEDIATE 'GRANT CREATE SESSION, CREATE TABLE, CREATE VIEW, CREATE SEQUENCE, CREATE PROCEDURE, CREATE TRIGGER, CREATE TYPE, CREATE SYNONYM TO %[1]s';
END;
/
`, name, password)
		}
		if u.App != u.Owner {
			fmt.Fprintf(&b, `GRANT SELECT ANY TABLE, INSERT ANY TABLE, UPDATE ANY TABLE, DELETE ANY TABLE, SELECT ANY SEQUENCE, EXECUTE ANY PROCEDURE TO %[1]s;
CREATE OR REPLACE TRIGGER %[1]s_LOGON AFTER LOGON ON %[1]s.SCHEMA
BEGIN EXECUTE IMMEDIATE 'ALTER SESSION SET CURRENT_SCHEMA = %[2]s'; END;
/
`, u.App, u.Owner)
		}
	}
	b.WriteString("EXIT\n")
	return b.String()
}

// CountSQL counts how many of the users exist; sqlplus exits 0 on empty
// input, so the result is checked against len(UserNames(users)) instead of
// trusting the exit code.
func CountSQL(users []DBUser) string {
	names := UserNames(users)
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "'" + n + "'"
	}
	return fmt.Sprintf("set heading off feedback off pages 0\nselect count(*) from dba_users where username in (%s);\nexit\n", strings.Join(quoted, ","))
}

func uniq(names ...string) []string {
	var out []string
	for _, n := range names {
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}
