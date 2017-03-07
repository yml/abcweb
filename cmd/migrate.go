package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kat-co/vala"
	"github.com/spf13/cobra"
)

var migrateCmdConfig migrateConfig

// migrateCmd represents the "migrate" command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run migration tasks (up, down, redo, status, version)",
	Long: `Run migration tasks on the migrations in your migrations directory.
Migrations can be generated by using the "abcweb generate migration" command.
This tool pipes out to Goose: https://github.com/pressly/goose
`,
	Example: "abcweb migrate up\nabcweb migrate down",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Migrate the DB to the most recent version available",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "up")
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the version by 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "down")
	},
}

var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Re-run the latest migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "redo")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Dump the migration status for the current DB",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "status")
	},
}

var dbVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version of the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "version")
	},
}

func init() {
	// migrate flags
	migrateCmd.PersistentFlags().StringP("db", "d", "", `Valid options: postgres|mysql (default: "database.toml db field")`)

	RootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(redoCmd)
	migrateCmd.AddCommand(statusCmd)
	migrateCmd.AddCommand(dbVersionCmd)

	migrateCmd.PersistentPreRun = func(*cobra.Command, []string) {
		cnf.ModeViper.BindPFlags(migrateCmd.PersistentFlags())
	}
}

func migrateExec(cmd *cobra.Command, args []string, subCmd string) error {
	checkDep("goose")

	err := cnf.CheckEnv()
	if err != nil {
		return err
	}

	migrateCmdConfig.DB = cnf.ModeViper.GetString("db")
	migrateCmdConfig.DBName = cnf.ModeViper.GetString("dbname")
	migrateCmdConfig.User = cnf.ModeViper.GetString("user")
	migrateCmdConfig.Pass = cnf.ModeViper.GetString("pass")
	migrateCmdConfig.Host = cnf.ModeViper.GetString("host")
	migrateCmdConfig.Port = cnf.ModeViper.GetInt("port")
	migrateCmdConfig.SSLMode = cnf.ModeViper.GetString("sslmode")

	var connStr string
	if migrateCmdConfig.DB == "postgres" {
		if migrateCmdConfig.SSLMode == "" {
			migrateCmdConfig.SSLMode = "require"
			cnf.ModeViper.Set("sslmode", migrateCmdConfig.SSLMode)
		}

		if migrateCmdConfig.Port == 0 {
			migrateCmdConfig.Port = 5432
			cnf.ModeViper.Set("port", migrateCmdConfig.Port)
		}
		connStr = postgresConnStr(migrateCmdConfig)
	} else if migrateCmdConfig.DB == "mysql" {
		if migrateCmdConfig.SSLMode == "" {
			migrateCmdConfig.SSLMode = "true"
			cnf.ModeViper.Set("sslmode", migrateCmdConfig.SSLMode)
		}

		if migrateCmdConfig.Port == 0 {
			migrateCmdConfig.Port = 3306
			cnf.ModeViper.Set("port", migrateCmdConfig.Port)
		}
		connStr = mysqlConnStr(migrateCmdConfig)
	}

	err = vala.BeginValidation().Validate(
		vala.StringNotEmpty(migrateCmdConfig.DB, "db"),
		vala.StringNotEmpty(migrateCmdConfig.User, "user"),
		vala.StringNotEmpty(migrateCmdConfig.Host, "host"),
		vala.Not(vala.Equals(migrateCmdConfig.Port, 0, "port")),
		vala.StringNotEmpty(migrateCmdConfig.DBName, "dbname"),
		vala.StringNotEmpty(migrateCmdConfig.SSLMode, "sslmode"),
	).Check()

	if err != nil {
		return err
	}

	excArgs := []string{
		migrateCmdConfig.DB,
		connStr,
		subCmd,
	}

	exc := exec.Command("goose", excArgs...)
	exc.Dir = filepath.Join(cnf.AppPath, "migrations")

	out, err := exc.CombinedOutput()

	fmt.Print(string(out))
	// On error exit here instead of return, because goose erroneously returns
	// error status codes on expected failures (like "no migration" errors),
	// which triggers the abcweb --help usage.
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return nil
}

// mysqlConnStr returns a mysql connection string compatible with the
// Go mysql driver package, in the format:
// user:pass@tcp(host:port)/dbname?tls=true
func mysqlConnStr(cfg migrateConfig) string {
	var out bytes.Buffer

	out.WriteString(cfg.User)
	if len(cfg.Pass) > 0 {
		out.WriteByte(':')
		out.WriteString(cfg.Pass)
	}
	out.WriteString(fmt.Sprintf("@tcp(%s:%d)/%s", cfg.Host, cfg.Port, cfg.DBName))
	if len(cfg.SSLMode) > 0 {
		out.WriteString("?tls=")
		out.WriteString(cfg.SSLMode)
	}

	return out.String()
}

// postgressConnStr returns a postgres connection string compatible with the
// Go pq driver package, in the format:
// user=bob password=secret host=1.2.3.4 port=5432 dbname=mydb sslmode=verify-full
func postgresConnStr(cfg migrateConfig) string {
	connStrs := []string{
		fmt.Sprintf("user=%s", cfg.User),
	}

	if len(cfg.Pass) > 0 {
		connStrs = append(connStrs, fmt.Sprintf("password=%s", cfg.Pass))
	}

	connStrs = append(connStrs, []string{
		fmt.Sprintf("host=%s", cfg.Host),
		fmt.Sprintf("port=%d", cfg.Port),
		fmt.Sprintf("dbname=%s", cfg.DBName),
	}...)

	if len(cfg.SSLMode) > 0 {
		connStrs = append(connStrs, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	return strings.Join(connStrs, " ")
}
