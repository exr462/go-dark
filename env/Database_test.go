package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/env"
)

func writeProps(t *testing.T, root, project, envName, body string) string {
	t.Helper()
	dir := filepath.Join(root, project)
	f := filepath.Join(dir, "helm", project, "config", envName, "application-k8s.properties")
	_ = os.MkdirAll(filepath.Dir(f), 0o755)
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestUsers_ReadsTheServicesConfig(t *testing.T) {
	root := t.TempDir()
	order := writeProps(t, root, "til-order", "dev", "spring.datasource.url=jdbc:x\nspring.datasource.username=TORA\nspring.liquibase.user=order_developer \n")
	dtm := writeProps(t, root, "til-dtm", "dev", "spring.datasource.username=tdtm\n")
	writeProps(t, root, "til-dtm", "prd", "spring.datasource.username=PRD_ONLY\n")
	plain := filepath.Join(root, "lib")
	_ = os.MkdirAll(plain, 0o755)

	users := env.Users([]string{order, dtm, plain, dtm}, "dev", []config.DatabaseUser{{App: "extra"}})

	want := []env.DBUser{{App: "EXTRA", Owner: "EXTRA"}, {App: "TDTM", Owner: "TDTM"}, {App: "TORA", Owner: "ORDER_DEVELOPER"}}
	if len(users) != len(want) {
		t.Fatalf("got %+v, want %+v", users, want)
	}
	for i := range want {
		if users[i] != want[i] {
			t.Fatalf("got %+v, want %+v", users, want)
		}
	}
	if got := strings.Join(env.UserNames(users), ","); got != "EXTRA,ORDER_DEVELOPER,TDTM,TORA" {
		t.Fatalf("UserNames = %s", got)
	}
}

func TestSchemaSQL_CreatesOwnersAppsGrantsAndTrigger(t *testing.T) {
	sql := env.SchemaSQL([]env.DBUser{{App: "TORA", Owner: "ORDER_DEVELOPER"}, {App: "TDTM", Owner: "TDTM"}}, "pw")

	if !strings.HasPrefix(sql, "WHENEVER SQLERROR EXIT FAILURE\n") || !strings.HasSuffix(sql, "EXIT\n") {
		t.Fatalf("sqlplus must stop on errors and exit:\n%s", sql)
	}
	for _, want := range []string{
		"WHERE username = 'ORDER_DEVELOPER'", `CREATE USER ORDER_DEVELOPER IDENTIFIED BY "pw" QUOTA UNLIMITED ON USERS`,
		"WHERE username = 'TORA'", "GRANT CREATE SESSION, CREATE TABLE, CREATE VIEW, CREATE SEQUENCE, CREATE PROCEDURE, CREATE TRIGGER, CREATE TYPE, CREATE SYNONYM TO TDTM",
		"GRANT SELECT ANY TABLE, INSERT ANY TABLE, UPDATE ANY TABLE, DELETE ANY TABLE, SELECT ANY SEQUENCE, EXECUTE ANY PROCEDURE TO TORA;",
		"CREATE OR REPLACE TRIGGER TORA_LOGON AFTER LOGON ON TORA.SCHEMA", "ALTER SESSION SET CURRENT_SCHEMA = ORDER_DEVELOPER",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q in:\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "TDTM_LOGON") || strings.Count(sql, "WHERE username = 'TDTM'") != 1 {
		t.Errorf("a service owning its schema gets one user and no trigger:\n%s", sql)
	}
	if count := env.CountSQL([]env.DBUser{{App: "TORA", Owner: "ORDER_DEVELOPER"}}); !strings.Contains(count, "where username in ('ORDER_DEVELOPER','TORA')") || !strings.HasSuffix(count, "exit\n") {
		t.Fatalf("count sql:\n%s", count)
	}
}
