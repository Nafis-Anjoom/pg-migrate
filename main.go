package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"pg-migrate/internal"
)

const CONFIG_LOCATION string = "./migrate.config"

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
    case "create":
        handleCreate(os.Args[2:])
	default:
		log.Println("invalid command")
		os.Exit(1)
	}
}

func handleCreate(args []string) {
    if len(args) != 1 {
        log.Fatal("Invalid create command. It only accepts one parameter: name of migration.")
        os.Exit(1)
    }

    config, err := parseConfig()
	if err != nil {
		log.Fatal("error opening migrate.config: ", err)
        os.Exit(1)
	}

    upstreamFileName := fmt.Sprintf("%s/%06d.%s.up.sql",config.MigrationSource, config.CurrentVersion + 1, args[0])
    downstreamFileName := fmt.Sprintf("%s/%06d.%s.down.sql", config.MigrationSource, config.CurrentVersion + 1, args[0])

    _, err = os.Create(upstreamFileName)
    if err != nil {
        log.Fatal("Error creating file:", err)
    }

    _, err = os.Create(downstreamFileName)
    if err != nil {
        os.Remove(upstreamFileName)
        log.Fatal("Error creating file:", err)
    }

    config.CurrentVersion += 1
	json, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		log.Fatal("error encoding migration config to json:", err)
	}

	err = os.WriteFile("./migrate.config", json, 0660)
	if err != nil {
		log.Fatal("error saving migration config file:", err)
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

// TODO: investigate if config flag is needed
// TODO: refactor to use parseConfig function
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

/* Utils */

func parseConfig() (*Config, error) {
	data, err := os.ReadFile(CONFIG_LOCATION)
	if err != nil {
		log.Fatal("error opening migrate.config: ", err)
        return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("invalid config file:", err)
        return nil, err
	}

    return &config, nil
}

func writeConfig(config *Config) error {
	json, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		log.Fatal("error encoding migration config to json:", err)
	}

	err = os.WriteFile("./migrate.config", json, 0660)
	if err != nil {
		log.Fatal("error creating migration config file:", err)
	}

    return nil
}
