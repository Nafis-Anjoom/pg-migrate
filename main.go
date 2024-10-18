package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"pg-migrate/internal"
	"strconv"
)

const CONFIG_LOCATION string = "./migrate.config"

type Config struct {
	CurrentVersion  int    `json:"current_version"`
	LatestVersion   int    `json:"latest_version"`
	DatabaseEnv     string `json:"conn_url"`
	MigrationSource string `json:"migrations_source"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("invalid command")
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
		fmt.Println("invalid command")
	}
}

func handleCreate(args []string) {
	if len(args) != 1 {
		fmt.Println("Invalid create command. It only accepts one parameter: name of migration.")
		return
	}

	config, err := parseConfig()
	if err != nil {
		fmt.Println("error opening migrate.config: ", err)
		return
	}

	config.LatestVersion += 1

	upstreamFileName := fmt.Sprintf("%s/%06d.%s.up.sql", config.MigrationSource, config.LatestVersion, args[0])
	downstreamFileName := fmt.Sprintf("%s/%06d.%s.down.sql", config.MigrationSource, config.LatestVersion, args[0])

	_, err = os.Create(upstreamFileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	_, err = os.Create(downstreamFileName)
	if err != nil {
		os.Remove(upstreamFileName)
		fmt.Println("Error creating file:", err)
		return
	}

	json, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		fmt.Println("error encoding migration config to json:", err)
		return
	}

	err = os.WriteFile("./migrate.config", json, 0660)
	if err != nil {
		fmt.Println("error saving migration config file:", err)
		return
	}
}

func handleInit(args []string) {
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	sourcePtr := flagSet.String("source", "./migrations", "Source directory for migrations. If the directory does not exist, the program will automatically create a new one.")
	databaseEnvPtr := flagSet.String("database", "", "Env variable for database connection string")
	flagSet.Parse(args)

	err := os.Mkdir(*sourcePtr, 0750)
	if errors.Is(err, fs.ErrExist) {
        fmt.Println("Could not initialize migration directory")
        fmt.Println("\t", err)
		return
	}

	connURL := os.Getenv(*databaseEnvPtr)
	if connURL == "" {
		fmt.Println("Database env variable is empty")
		return
	}

	var config Config
	config.MigrationSource = *sourcePtr
	config.DatabaseEnv = *databaseEnvPtr

	err = writeConfig(&config)
	if err != nil {
		fmt.Println("Error encoding migration config to json:", err)
		return
	}

	err = internal.InitVersionTable(connURL)
	if err != nil {
		os.Remove("./migrate.config")
		fmt.Println("Error initializing version table:", err)
		return
	}

	fmt.Println("Successfully initialized migraitons directory:", *sourcePtr)
	fmt.Println("Successfully created version table: versionTable")
	fmt.Println("To get Started, edit the migrate.config file in the current directory")
}

// TODO: investigate if config flag is needed
// TODO: refactor to use parseConfig function
func handleMigrate(args []string) {
	flagSet := flag.NewFlagSet("migrate", flag.ExitOnError)
    targetPtr := flagSet.String("target", "latest", "migration target")
	flagSet.Parse(args)

    config, err := parseConfig()
	if err != nil {
		fmt.Println("error opening migrate.config: ", err)
		return
	}

    var direction internal.Direction
    var end int

    if *targetPtr == "latest" {
        end = config.LatestVersion
        direction = internal.UP
    } else {
        if target := *targetPtr; target[0] == '-' {
            num, err := strconv.ParseInt(target[1:], 10, 64)
            if err != nil {
                fmt.Println("invalid target migration:", target)
                return
            }
            direction = internal.DOWN
            end = config.CurrentVersion - int(num)
        } else if target[0] == '+' {
            num, err := strconv.ParseInt(target[1:], 10, 64)
            if err != nil {
                fmt.Println("invalid target migration:", target)
                return
            }
            direction = internal.UP
            end = config.CurrentVersion + int(num)
        } else {
            num, err := strconv.ParseInt(target, 10, 64)
            if err != nil {
                fmt.Println("invalid target migration version:", target)
                return
            }

            if num == int64(config.CurrentVersion) {
                fmt.Println("schema already at version", num)
                return
            } else if num > int64(config.CurrentVersion) {
                direction = internal.UP
                end = int(num)
            } else {
                direction = internal.DOWN
                end = int(num)
            }
        }
    }

    if end < 0 || end > config.LatestVersion {
        fmt.Println("invalid target migration version:", end)
        return
    }

	connURL := os.Getenv(config.DatabaseEnv)
	if connURL == "" {
		fmt.Println("missing database url")
		return
	}

    // TODO: optimize NewMigrater by parsing only the ones denoted by the start and end paramters
	migrater, err := internal.NewMigrater(config.MigrationSource, connURL)
	if err != nil {
		fmt.Println("error creating migrater:", err)
		return
	}

    var migrationErr error
    if direction == internal.UP {
        migrationErr = migrater.RunUpstreamMigrations(config.CurrentVersion, end)
    } else {
        migrationErr = migrater.RunDownstreamMigrations(config.CurrentVersion, end)
    }

    if migrationErr != nil {
        fmt.Println("Migration error")
        fmt.Println("\t", migrationErr)
    }

    config.CurrentVersion = end
    err = writeConfig(config)
    if err != nil {
        fmt.Println("Error writing config file:", err)
    }
}

/* Utils */

func parseConfig() (*Config, error) {
	data, err := os.ReadFile(CONFIG_LOCATION)
	if err != nil {
		return nil, fmt.Errorf("error opening migrate.config: %w", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &config, nil
}

func writeConfig(config *Config) error {
	json, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("error encoding migration config to json: %w", err)
	}

	err = os.WriteFile("./migrate.config", json, 0660)
	if err != nil {
		return fmt.Errorf("error creating migration config file: %w", err)
	}

	return nil
}
