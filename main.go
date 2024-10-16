package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"pg-migrate/internal"
)

type Config struct {
	CurrentVersion  int    `json:"current_version"`
	DatabaseEnv     string `json:"conn_url"`
	MigrationSource string `json:"migrations_source"`
}

func main() {
	if len(os.Args) < 2 {
		log.Println("invalid command")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		handleMigrate(os.Args[2:])
	case "init":
		handleInit(os.Args[2:])
	default:
		log.Println("invalid command")
		os.Exit(1)
	}
}

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	sourcePtr := fs.String("source", "./migrations", "Source directory for migrations. If the directory does not exist, the program will automatically create a new one.")
	databaseEnvPtr := fs.String("databaseEnv", "", "env variable for database connection string")
	fs.Parse(args)

	err := os.Mkdir(*sourcePtr, 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal("error initializing migration directory:", err)
	}

    connURL := os.Getenv(*databaseEnvPtr)
    if connURL == "" {
        log.Fatal("database env variable is empty")
    }

	var config Config
	config.MigrationSource = *sourcePtr
	// config.VersionTable = *versionTablePtr
	config.DatabaseEnv = *databaseEnvPtr

	json, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		log.Fatal("error encoding migration config to json:", err)
	}

	err = os.WriteFile("./migrate.config", json, 0660)
	if err != nil {
		log.Fatal("error creating migration config file:", err)
	}

    err = internal.InitVersionTable(connURL)
    if err != nil {
        // instead of doing transaction, we simply revert the write file operation
        os.Remove("./migrate.config")
        log.Fatal("error initializing version table")
    }

	log.Println("successfully initialized migraitons directory:", *sourcePtr)
	log.Println("successfully created version table: versionTable")
	log.Println(`To get Started, edit the migrate.config file in the current directory. If using env variable, then pass "$ENV_VARIABLE"`)
}

func handleMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	configSrcPtr := flag.String("config", "./migrate.config", "config file source")
	fs.Parse(args)

	data, err := os.ReadFile(*configSrcPtr)
	if err != nil {
		log.Fatal("error opening migrate.config: ", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("invalid config file:", err)
	}

    connURL := os.Getenv(config.DatabaseEnv)

	if connURL == "" {
		log.Fatal("missing database url")
	}

	migrater, err := internal.NewMigrater(config.MigrationSource, connURL)
	if err != nil {
		log.Fatal("error creating migrater")
	}

	migrater.RunMigrations(internal.UP)
}
